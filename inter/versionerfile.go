package inter

import (
	"context"

	"solidcoredata.org/src/databus/bus"
)

var _ BusVersioner = &FileBus{}
var _ BusReader = &FileBus{}

func NewFileBus(projectRoot string) (*FileBus, error) {
	return &FileBus{}, nil
}

type FileBus struct{}

func (fb *FileBus) List(ctx context.Context) ([]bus.Version, error) {
	panic("TODO")
}
func (fb *FileBus) Get(ctx context.Context, bv bus.Version) (*bus.Bus, error) {
	panic("TODO")
}
func (fb *FileBus) Amend(ctx context.Context, existing bus.Version, b *bus.Bus) (bus.Version, error) {
	panic("TODO")
}
func (fb *FileBus) Commit(ctx context.Context, b *bus.Bus) (bus.Version, error) {
	panic("TODO")
}

func (fb *FileBus) GetBus(ctx context.Context) (*bus.Bus, error) {
	panic("TODO")
}
