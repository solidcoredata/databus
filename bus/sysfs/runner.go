package sysfs

import (
	"context"

	"solidcoredata.org/src/databus/bus"
)

var _ bus.Runner = &runner{}

func NewRunner(root string) bus.Runner {
	return &runner{
		root: root,
	}
}

type runner struct {
	root string
}

/*
type Project struct {
	// This could be a filesystem root or an HTTP root path.
	Root     string
	Enteries []RunnerEntry
}
type RunnerEntry struct {
	Call    string
	Options map[string]string
}
*/

func (r *runner) Run(ctx context.Context, setup bus.Project, vdelta bus.VersionDelta, currentBus *bus.Bus, deltaBus interface{}) error {
	for _, e := range setup.Enteries {
		_ = e
		// TODO(daniel.theophanes): assume Call path is an exec, not an http call. Call exec.
	}
	return nil
}
