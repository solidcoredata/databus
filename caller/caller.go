package caller

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

func (c *SimpleCaller) listExt(ctx context.Context) ([]Extension, error) {
	exts, err := c.extReg.List(ctx)
	if err != nil {
		return nil, err
	}
	ret := make([]Extension, len(exts))
	for i, extName := range exts {
		ext, err := c.extReg.Get(ctx, extName)
		if err != nil {
			return nil, err
		}
		ret[i] = ext
	}
	return ret, nil
}

func (c *SimpleCaller) Validate(ctx context.Context) error {
	b, err := c.busRead.GetBus(ctx)
	if err != nil {
		return err
	}
	exts, err := c.listExt(ctx)
	if err != nil {
		return err
	}
	for _, ext := range exts {
		about := ext.AboutSelf()
		err = ext.Validate(ctx, b.Filter(about.HandleTypes))
		if err != nil {
			return err
		}
	}
	return b.Init()
}

func (c *SimpleCaller) currentPrevious(ctx context.Context, src bool) (current *bus.Bus, previous *bus.Bus, exts []Extension, err error) {
	var b1, b2 *bus.Bus
	if src {
		b1, err = c.busRead.GetBus(ctx)
		if err != nil {
			return nil, nil, nil, err
		}
		b2, err = c.busVersion.Get(ctx, bus.Version{Sequence: 0})
		if err != nil {
			return nil, nil, nil, err
		}
	} else {
		b1, err = c.busVersion.Get(ctx, bus.Version{Sequence: 0})
		if err != nil {
			return nil, nil, nil, err
		}
		b2, err = c.busVersion.Get(ctx, bus.Version{Sequence: -1})
		if err != nil {
			return nil, nil, nil, err
		}
	}
	exts, err = c.listExt(ctx)
	if err != nil {
		return nil, nil, nil, err
	}
	for _, ext := range exts {
		about := ext.AboutSelf()
		err = ext.Validate(ctx, b1.Filter(about.HandleTypes))
		if err != nil {
			return nil, nil, nil, err
		}
		err = ext.Validate(ctx, b2.Filter(about.HandleTypes))
		if err != nil {
			return nil, nil, nil, err
		}
	}
	return b1, b2, exts, nil
}

func (c *SimpleCaller) Diff(ctx context.Context, src bool) (*bus.DeltaBus, error) {
	current, previous, _, err := c.currentPrevious(ctx, src)
	if err != nil {
		return nil, err
	}
	return bus.NewDelta(current, previous)
}

func (c *SimpleCaller) Commit(ctx context.Context, amend bool) (bus.Version, error) {
	b, err := c.busRead.GetBus(ctx)
	if err != nil {
		return bus.Version{}, err
	}
	exts, err := c.listExt(ctx)
	if err != nil {
		return bus.Version{}, err
	}
	for _, ext := range exts {
		about := ext.AboutSelf()
		err = ext.Validate(ctx, b.Filter(about.HandleTypes))
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
	current, previous, exts, err := c.currentPrevious(ctx, src)
	if err != nil {
		return err
	}

	for _, ext := range exts {
		about := ext.AboutSelf()
		diff, err := bus.NewDelta(current.Filter(about.HandleTypes), previous.Filter(about.HandleTypes))
		if err != nil {
			return err
		}
		err = ext.Generate(ctx, diff, func(ctx context.Context, path string, content []byte) error {
			return c.extReadWrite.Put(ctx, about.Name, current.Version, path, content)
		})
		if err != nil {
			return err
		}
	}
	return nil
}
func (c *SimpleCaller) Deploy(ctx context.Context, src bool, opts *DeployOptions) error {
	current, previous, exts, err := c.currentPrevious(ctx, src)
	if err != nil {
		return err
	}

	for _, ext := range exts {
		about := ext.AboutSelf()
		diff, err := bus.NewDelta(current.Filter(about.HandleTypes), previous.Filter(about.HandleTypes))
		if err != nil {
			return err
		}
		err = ext.Deploy(ctx, opts, diff, func(ctx context.Context, path string) ([]byte, error) {
			return c.extReadWrite.Get(ctx, about.Name, current.Version, path)
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
