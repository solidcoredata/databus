package sysfs

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"

	"solidcoredata.org/src/databus/bus"
	"solidcoredata.org/src/databus/bus/load"
)

var _ bus.Input = &input{}

const (
	InputFilename  = "bus.jsonnet"
	ConfigFilename = "scd.jsonnet"
	InputDir       = "src"

	versionFilename = "bus.json"
	deltaFilename   = "delta.json"
	versionDir      = "version"
)

type input struct {
	root string
}

func NewInput(root string) bus.Input {
	return &input{
		root: root,
	}
}

// ReadBus reads the given bus version from either the src directory or
// the version directory.
func (i *input) ReadBus(ctx context.Context, opts bus.InputOptions) (*bus.Bus, error) {
	p := ""
	v := bus.Version{}
	switch {
	case opts.Src:
		p = filepath.Join(i.root, InputDir, InputFilename)
	case opts.Version == 0:
		list, err := i.ListVersion(ctx)
		if err != nil {
			return nil, err
		}
		if len(list) == 0 {
			return nil, fmt.Errorf("bus/sysfs: cannot read current version, no current version exists")
		}
		v = list[len(list)-1]
		p = filepath.Join(i.root, versionDir, strconv.FormatInt(v.Version, 10), versionFilename)
	case opts.Version < 0:
		list, err := i.ListVersion(ctx)
		if err != nil {
			return nil, err
		}
		nFromCurrent := -opts.Version
		if int64(len(list)) < nFromCurrent+1 {
			return nil, fmt.Errorf("bus/sysfs: cannot read %d version, requested version does not exists", opts.Version)
		}
		v = list[int64(len(list))-1-nFromCurrent]
		p = filepath.Join(i.root, versionDir, strconv.FormatInt(v.Version, 10), versionFilename)
	case opts.Version > 0:
		v.Version = opts.Version
		p = filepath.Join(i.root, versionDir, strconv.FormatInt(opts.Version, 10), versionFilename)
	default:
		panic("can't happen")
	}
	bus, err := load.Bus(ctx, p)
	if err != nil {
		return nil, err
	}
	bus.Version = v
	return bus, nil
}
func (i *input) ListVersion(ctx context.Context) ([]bus.Version, error) {
	vdir := filepath.Join(i.root, versionDir)
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
		vv = append(vv, bus.Version{Version: n})
	}
	sort.Slice(vv, func(i, j int) bool {
		return vv[i].Version < vv[j].Version
	})
	return vv, nil
}

func (i *input) ReadProject(ctx context.Context) (bus.Project, error) {
	p := filepath.Join(i.root, ConfigFilename)
	proj := bus.Project{}
	err := load.Decode(ctx, p, &proj)
	return proj, err
}
