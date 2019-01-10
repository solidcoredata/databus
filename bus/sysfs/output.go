package sysfs

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strconv"

	"solidcoredata.org/src/databus/bus"
)

var _ bus.Output = &output{}

func NewOutput(root string) bus.Output {
	return &output{
		root: root,
	}
}

type output struct {
	root string
}

// WriteBus writes the bus into the version folder.
func (o *output) WriteBus(ctx context.Context, ver bus.Version, currentBus *bus.Bus) error {
	vdir := filepath.Join(o.root, versionDir, strconv.FormatInt(ver.Version, 10))
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
	return coder.Encode(currentBus)
}

// WriteDelta writes the delta into the current version.
func (o *output) WriteDelta(ctx context.Context, vdelta bus.VersionDelta, deltaBus bus.DeltaBus) error {
	vdir := filepath.Join(o.root, versionDir, strconv.FormatInt(vdelta.Current.Version, 10))
	if err := os.MkdirAll(vdir, 0700); err != nil {
		return err
	}
	filename := filepath.Join(vdir, deltaFilename)
	f, err := os.OpenFile(filename, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0600)
	if err != nil {
		return err
	}
	defer f.Close()
	coder := json.NewEncoder(f)
	coder.SetEscapeHTML(false)
	coder.SetIndent("", "\t")
	return coder.Encode(deltaBus)
}
