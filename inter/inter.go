// Package inter is the second take on the CLI interface.
package inter

import (
	"context"

	"solidcoredata.org/src/databus/bus"
)

// Caller represents the CLI tool right now.
// This should probably be a struct, not an interface.
// The caller will provide a few abstractions:
//  * How to read the Bus definition. Almost certainly fetch from local disk for a long time.
//  * How to read, write, and list the bus and delta bus versions. This could be local disk or object storage.
//  * How to read and write extension files (disk, memory, object storage).
//  * How to register and call extensions.
//    - The first extensions will be compiled into the main CLI.
//    - Later it could be changed to a plugin system like github.com/hashicorp/go-plugin.
type Caller interface {
	Validate()
	Diff()
	Commit()
	Generate()
	Deploy()
	UI()
}

type ExtensionAbout struct {
	HandleTypes []string
}

type Extension interface {
	// Return information what this extension can handle and do.
	AboutSelf() (ExtensionAbout, error)

	// Extension specific Bus validation.
	Validate(ctx context.Context, b *bus.Bus) error

	// Generate and write files. Note, no file list is provided so extensions should
	// write a manafest file of some type by a well known name.
	Generate(ctx context.Context, writeFile ExtensionVersionWriter) error

	// Read generated files and deploy to system.
	Deploy(ctx context.Context, readFile ExtensionVersionReader) error
}

// Read the current, in-progress definition.
type BusReader interface {
	GetBus(ctx context.Context) (*bus.Bus, error)
}

// Read or write the bus or delta bus definitions at a given version.
type BusVersioner interface {
	List(ctx context.Context) ([]bus.Version, error)
	Get(ctx context.Context, v bus.Version) (*bus.Bus, error)
	Ammend(ctx context.Context, existing bus.Version, b *bus.Bus) (bus.Version, error)
	Commit(ctx context.Context, b *bus.Bus) (bus.Version, error)
}

// Read or write a file within an extension context and bus version.
type ExtensionReadWriter interface {
	Get(ctx context.Context, extname string, busVersion bus.Version, path string) ([]byte, error)
	Put(ctx context.Context, extname string, busVersion bus.Version, path string, content []byte) error
}

// Used by an extension to read a given file.
type ExtensionVersionReader interface {
	Get(ctx context.Context, path string) ([]byte, error)
}

// Used by an extension to write a given file.
type ExtensionVersionWriter interface {
	Put(ctx context.Context, path string, content []byte) error
}

type ExtensionRegister interface {
	List() ([]string, error)
	Get(name string) (Extension, error)
}

type CallerSetup struct {
	BusReader
	BusVersioner
	ExtensionReadWriter
	ExtensionRegister
}

func NewCaller(setup CallerSetup) (Caller, error) {
	panic("TODO")
}
