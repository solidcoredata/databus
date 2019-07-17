package inter

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"

	"solidcoredata.org/src/databus/bus"
	"solidcoredata.org/src/databus/bus/load"
)

const (
	InputFilename  = "bus.jsonnet"
	ConfigFilename = "scd.jsonnet"
	InputDir       = "src"

	versionFilename = "bus.json"
	deltaFilename   = "delta.json"
	versionDir      = "version"
)

var _ BusVersioner = &FileBus{}
var _ BusReader = &FileBus{}

func NewFileBus(projectRoot string) (*FileBus, error) {
	return &FileBus{
		root: projectRoot,
	}, nil
}

type FileBus struct {
	root string
}

func (fb *FileBus) List(ctx context.Context) ([]bus.Version, error) {
	vdir := filepath.Join(fb.root, versionDir)
	v, err := os.Open(vdir)
	if v != nil {
		defer v.Close()
	}
	if os.IsNotExist(err) {
		// No versions released yet.
		return []bus.Version{}, nil
	}
	if err != nil {
		return nil, err
	}

	list, err := v.Readdir(-1)
	if err != nil {
		return nil, err
	}
	vv := make([]bus.Version, 0, len(list))
	for _, item := range list {
		if !item.IsDir() {
			continue
		}
		n, err := strconv.ParseInt(item.Name(), 10, 64)
		if err != nil {
			continue
		}
		vv = append(vv, bus.Version{Sequence: n})
	}
	sort.Slice(vv, func(i, j int) bool {
		return vv[i].Sequence < vv[j].Sequence
	})
	return vv, nil
}
func (fb *FileBus) Get(ctx context.Context, bv bus.Version) (*bus.Bus, error) {
	p := ""
	switch {
	case bv.Sequence == 0:
		list, err := fb.List(ctx)
		if err != nil {
			return nil, err
		}
		if len(list) == 0 {
			return nil, fmt.Errorf("inter: cannot read current version, no current version exists")
		}
		v := list[len(list)-1]
		p = filepath.Join(fb.root, versionDir, strconv.FormatInt(v.Sequence, 10), versionFilename)
	case bv.Sequence < 0:
		list, err := fb.List(ctx)
		if err != nil {
			return nil, err
		}
		nFromCurrent := -bv.Sequence
		if int64(len(list)) < nFromCurrent+1 {
			return nil, fmt.Errorf("inter: cannot read %d version, requested version does not exists", bv.Sequence)
		}
		v := list[int64(len(list))-1-nFromCurrent]
		p = filepath.Join(fb.root, versionDir, strconv.FormatInt(v.Sequence, 10), versionFilename)
	case bv.Sequence > 0:
		p = filepath.Join(fb.root, versionDir, strconv.FormatInt(bv.Sequence, 10), versionFilename)
	default:
		panic("can't happen")
	}
	bus, err := load.Bus(ctx, p)
	if err != nil {
		return nil, err
	}
	return bus, nil
}
func (fb *FileBus) Amend(ctx context.Context, existing bus.Version, b *bus.Bus) (bus.Version, error) {
	x := *b
	x.Version = existing
	err := fb.writeBus(ctx, &x)
	return x.Version, err
}
func (fb *FileBus) Commit(ctx context.Context, b *bus.Bus) (bus.Version, error) {
	var v bus.Version
	list, err := fb.List(ctx)
	if err != nil {
		return v, err
	}
	if len(list) > 0 {
		v = list[len(list)-1]
	}
	x := *b
	x.Version = bus.Version{Sequence: v.Sequence + 1}
	err = fb.writeBus(ctx, &x)
	return x.Version, err
}

func (fb *FileBus) writeBus(ctx context.Context, b *bus.Bus) error {
	vdir := filepath.Join(fb.root, versionDir, strconv.FormatInt(b.Version.Sequence, 10))
	if err := os.MkdirAll(vdir, 0700); err != nil {
		return err
	}
	filename := filepath.Join(vdir, versionFilename)
	f, err := os.OpenFile(filename, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0600)
	if err != nil {
		return err
	}
	defer f.Close()
	coder := json.NewEncoder(f)
	coder.SetEscapeHTML(false)
	coder.SetIndent("", "\t")
	return coder.Encode(b)
}

func (fb *FileBus) GetBus(ctx context.Context) (*bus.Bus, error) {
	p := filepath.Join(fb.root, InputDir, InputFilename)
	bus, err := load.Bus(ctx, p)
	if err != nil {
		return nil, err
	}
	return bus, nil
}
