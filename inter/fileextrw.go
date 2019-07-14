package inter

import (
	"context"

	"solidcoredata.org/src/databus/bus"
)

var _ ExtensionReadWriter = &FileExtRW{}

func NewFileExtRW(projectRoot string) (*FileExtRW, error) {
	return &FileExtRW{}, nil
}

type FileExtRW struct{}

func (f *FileExtRW) Get(ctx context.Context, extname string, busVersion bus.Version, path string) ([]byte, error) {
	panic("TODO")
}
func (f *FileExtRW) Put(ctx context.Context, extname string, busVersion bus.Version, path string, content []byte) error {
	panic("TODO")
}
