// Package inter is the second take on the CLI interface.
package inter

import (
	"context"

	"solidcoredata.org/src/databus/bus"
)

type ExtensionAbout struct {
	Name        string
	HandleTypes []string
}

type Extension interface {
	// Return information what this extension can handle and do.
	AboutSelf() ExtensionAbout

	// Extension specific Bus validation.
	Validate(ctx context.Context, b *bus.Bus) error

	// Generate and write files. Note, no file list is provided so extensions should
	// write a manafest file of some type by a well known name.
	Generate(ctx context.Context, diff *bus.DeltaBus, writeFile ExtensionVersionWriter) error

	// Read generated files and deploy to system.
	Deploy(ctx context.Context, opts *DeployOptions, diff *bus.DeltaBus, readFile ExtensionVersionReader) error
}

type DeployOptions struct {
	EnvironmentName   string   // Environment name to deploy to.
	CreateEnvironment bool     // Create an environment if none found with name.
	RunTasks          []string // Run these tasks within the environment.

	// Delete the environment when finished.
	// When true and Create is false and tasks are empty, looks for and deletes environment only.
	DeleteEnvironment bool
}

// Read the current, in-progress definition.
type BusReader interface {
	GetBus(ctx context.Context) (*bus.Bus, error)
}

// Read or write the bus or delta bus definitions at a given version.
type BusVersioner interface {
	List(ctx context.Context) ([]bus.Version, error)
	Get(ctx context.Context, v bus.Version) (*bus.Bus, error)
	Amend(ctx context.Context, existing bus.Version, b *bus.Bus) (bus.Version, error)
	Commit(ctx context.Context, b *bus.Bus) (bus.Version, error)
}

// Read or write a file within an extension context and bus version.
type ExtensionReadWriter interface {
	Get(ctx context.Context, extname string, busVersion bus.Version, path string) ([]byte, error)
	Put(ctx context.Context, extname string, busVersion bus.Version, path string, content []byte) error
}

// Used by an extension to read a given file.
type ExtensionVersionReader func(ctx context.Context, path string) ([]byte, error)

// Used by an extension to write a given file.
type ExtensionVersionWriter func(ctx context.Context, path string, content []byte) error

type ExtensionRegister interface {
	List(ctx context.Context) ([]string, error)
	Get(ctx context.Context, name string) (Extension, error)
}
