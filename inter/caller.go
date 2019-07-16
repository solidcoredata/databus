package inter

import (
	"context"

	"solidcoredata.org/src/databus/bus"
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
type SimpleCaller struct {
	busRead      BusReader
	busVersion   BusVersioner
	extReadWrite ExtensionReadWriter
	extReg       ExtensionRegister
}

type CallerSetup struct {
	Bus       BusReader
	Versioner BusVersioner
	ExtRW     ExtensionReadWriter
	ExtReg    ExtensionRegister
}

func NewCaller(setup CallerSetup) (*SimpleCaller, error) {
	return &SimpleCaller{
		busRead:      setup.Bus,
		busVersion:   setup.Versioner,
		extReadWrite: setup.ExtRW,
		extReg:       setup.ExtReg,
	}, nil
}

func (c *SimpleCaller) Validate(ctx context.Context) error {
	b, err := c.busRead.GetBus(ctx)
	if err != nil {
		return err
	}
	exts, err := c.extReg.List(ctx)
	if err != nil {
		return err
	}
	for _, extName := range exts {
		ext, err := c.extReg.Get(ctx, extName)
		if err != nil {
			return err
		}
		err = ext.Validate(ctx, b)
		if err != nil {
			return err
		}
	}
	return b.Init()
}
func (c *SimpleCaller) Diff(ctx context.Context, src bool) (*bus.DeltaBus, error) {
	var err error
	var b1, b2 *bus.Bus
	if src {
		b1, err = c.busRead.GetBus(ctx)
		if err != nil {
			return nil, err
		}
		b2, err = c.busVersion.Get(ctx, bus.Version{Sequence: 0})
		if err != nil {
			return nil, err
		}
	} else {
		b1, err = c.busVersion.Get(ctx, bus.Version{Sequence: 0})
		if err != nil {
			return nil, err
		}
		b2, err = c.busVersion.Get(ctx, bus.Version{Sequence: -1})
		if err != nil {
			return nil, err
		}
	}
	err = b1.Init()
	if err != nil {
		return nil, err
	}
	err = b2.Init()
	if err != nil {
		return nil, err
	}
	exts, err := c.extReg.List(ctx)
	if err != nil {
		return nil, err
	}
	for _, extName := range exts {
		ext, err := c.extReg.Get(ctx, extName)
		if err != nil {
			return nil, err
		}
		err = ext.Validate(ctx, b1)
		if err != nil {
			return nil, err
		}
		err = ext.Validate(ctx, b2)
		if err != nil {
			return nil, err
		}
	}
	return bus.NewDelta(b1, b2)
}
func (c *SimpleCaller) Commit(ctx context.Context, amend bool) (bus.Version, error) {
	b, err := c.busRead.GetBus(ctx)
	if err != nil {
		return bus.Version{}, err
	}
	err = b.Init()
	if err != nil {
		return bus.Version{}, err
	}
	exts, err := c.extReg.List(ctx)
	if err != nil {
		return bus.Version{}, err
	}
	for _, extName := range exts {
		ext, err := c.extReg.Get(ctx, extName)
		if err != nil {
			return bus.Version{}, err
		}
		err = ext.Validate(ctx, b)
		if err != nil {
			return bus.Version{}, err
		}
	}
	if amend {
		return c.busVersion.Amend(ctx, bus.Version{Sequence: 0}, b)
	}
	return c.busVersion.Commit(ctx, b)
}

func (c *SimpleCaller) Generate(ctx context.Context, src bool) error {
	var err error
	var b *bus.Bus
	if src {
		b, err = c.busRead.GetBus(ctx)
	} else {
		b, err = c.busVersion.Get(ctx, bus.Version{Sequence: 0})
	}
	if err != nil {
		return err
	}
	err = b.Init()
	if err != nil {
		return err
	}

	exts, err := c.extReg.List(ctx)
	if err != nil {
		return err
	}
	for _, extName := range exts {
		ext, err := c.extReg.Get(ctx, extName)
		if err != nil {
			return err
		}
		err = ext.Validate(ctx, b)
		if err != nil {
			return err
		}
		err = ext.Generate(ctx, b, func(ctx context.Context, path string, content []byte) error {
			return c.extReadWrite.Put(ctx, extName, b.Version, path, content)
		})
		if err != nil {
			return err
		}
	}
	return nil
}
func (c *SimpleCaller) Deploy(ctx context.Context, src bool, opts *DeployOptions) error {
	var err error
	var b *bus.Bus
	if src {
		b, err = c.busRead.GetBus(ctx)
	} else {
		b, err = c.busVersion.Get(ctx, bus.Version{Sequence: 0})
	}
	if err != nil {
		return err
	}
	err = b.Init()
	if err != nil {
		return err
	}

	exts, err := c.extReg.List(ctx)
	if err != nil {
		return err
	}
	for _, extName := range exts {
		ext, err := c.extReg.Get(ctx, extName)
		if err != nil {
			return err
		}
		err = ext.Validate(ctx, b)
		if err != nil {
			return err
		}
		err = ext.Deploy(ctx, opts, b, func(ctx context.Context, path string) ([]byte, error) {
			return c.extReadWrite.Get(ctx, extName, b.Version, path)
		})
		if err != nil {
			return err
		}
	}
	return nil
}
func (c *SimpleCaller) UI(ctx context.Context) error {
	panic("TODO")
}
