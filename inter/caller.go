package inter

import (
	"context"
)

// SimpleCaller represents the CLI tool right now.
// This should probably be a struct, not an interface.
// The caller will provide a few abstractions:
//  * How to read the Bus definition. Almost certainly fetch from local disk for a long time.
//  * How to read, write, and list the bus and delta bus versions. This could be local disk or object storage.
//  * How to read and write extension files (disk, memory, object storage).
//  * How to register and call extensions.
//    - The first extensions will be compiled into the main CLI.
//    - Later it could be changed to a plugin system like github.com/hashicorp/go-plugin.
type SimpleCaller struct{}

type CallerSetup struct {
	Bus       BusReader
	Versioner BusVersioner
	ExtRW     ExtensionReadWriter
	ExtReg    ExtensionRegister
}

func NewCaller(setup CallerSetup) (*SimpleCaller, error) {
	panic("TODO")
}

func (c *SimpleCaller) Validate(ctx context.Context) error {
	panic("TODO")
}
func (c *SimpleCaller) Diff(ctx context.Context, src bool) error {
	panic("TODO")
}
func (c *SimpleCaller) Commit(ctx context.Context) error {
	panic("TODO")
}
func (c *SimpleCaller) Generate(ctx context.Context, src bool) error {
	panic("TODO")
}
func (c *SimpleCaller) Deploy(ctx context.Context, src bool) error {
	panic("TODO")
}
func (c *SimpleCaller) UI(ctx context.Context) error {
	panic("TODO")
}
