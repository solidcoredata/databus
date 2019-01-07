package sysfs

import (
	"context"
	"path/filepath"

	"solidcoredata.org/src/databus/bus"
	"solidcoredata.org/src/databus/bus/load"
)

var _ bus.Input = &input{}

const (
	inputFilename = "bus.jsonnet"
	inputDir      = "src"
	versionDir    = "version"
)

type input struct {
	root string
}

func NewInput(root string) bus.Input {
	return &input{
		root: root,
	}
}

func (i *input) ReadBus(ctx context.Context, opts bus.InputOptions) (*bus.Bus, error) {
	p := filepath.Join(i.root, inputDir, inputFilename)
	return load.Bus(ctx, p)
}
func (i *input) ListVersion(ctx context.Context) ([]bus.Version, error) {
	panic("todo")
}
