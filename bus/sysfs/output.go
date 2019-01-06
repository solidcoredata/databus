package sysfs

import (
	"context"

	"solidcoredata.org/src/databus/bus"
)

var _ bus.Output = &output{}

func NewOutput(root string) bus.Output {
	return &output{}
}

type output struct{}

func (o *output) WriteBus(ctx context.Context, ver bus.Version, currentBus *bus.Bus) error {
	panic("todo")
}
func (o *output) WriteDelta(ctx context.Context, vdelta bus.VersionDelta, deltaBus interface{}) error {
	panic("todo")
}
