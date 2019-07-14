package inter

import (
	"context"

	"solidcoredata.org/src/databus/bus"
)

func NewCRDB() *CRDB {
	return &CRDB{}
}

var _ Extension = &CRDB{}

type CRDB struct{}

func (cr *CRDB) AboutSelf(ctx context.Context) (ExtensionAbout, error) {
	return ExtensionAbout{}, nil
}

// Extension specific Bus validation.
func (cr *CRDB) Validate(ctx context.Context, b *bus.Bus) error {
	return nil
}

// Generate and write files. Note, no file list is provided so extensions should
// write a manafest file of some type by a well known name.
func (cr *CRDB) Generate(ctx context.Context, writeFile ExtensionVersionWriter) error {
	return nil
}

// Read generated files and deploy to system.
func (cr *CRDB) Deploy(ctx context.Context, readFile ExtensionVersionReader) error {
	return nil
}
