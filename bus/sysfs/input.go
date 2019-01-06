package sysfs

import (
	"context"

	"solidcoredata.org/src/databus/bus"
)

var _ bus.Input = &input{}

type input struct{}

func NewInput(root string) bus.Input {
	return &input{}
}

func (i *input) ReadBus(ctx context.Context, opts bus.InputOptions) (*bus.Bus, error) {
	panic("todo")
}
func (i *input) ListVersion(ctx context.Context) ([]bus.Version, error) {
	panic("todo")
}
