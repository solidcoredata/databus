package sysfs

import (
	"context"

	"solidcoredata.org/src/databus/bus"
)

var _ bus.Runner = &runner{}

func NewRunner(root string) bus.Runner {
	return &runner{}
}

type runner struct{}

func (r *runner) RunnerSetup(ctx context.Context) (bus.RunnerSetup, error) {
	panic("todo")
}
func (r *runner) Run(ctx context.Context, setup bus.RunnerSetup, vdelta bus.VersionDelta, currentBus *bus.Bus, deltaBus interface{}) error {
	panic("todo")
}
