package caller

import (
	"context"
	"io/ioutil"
	"path/filepath"
	"strconv"

	"solidcoredata.org/src/databus/bus"
)

var _ ExtensionReadWriter = &FileExtRW{}

func NewFileExtRW(projectRoot string) (*FileExtRW, error) {
	return &FileExtRW{}, nil
}

type FileExtRW struct {
	root string
}

func (f *FileExtRW) Get(ctx context.Context, extname string, busVersion bus.Version, path string) ([]byte, error) {
	path = filepath.Clean(path)
	full := filepath.Join(f.root, "ext", strconv.FormatInt(busVersion.Sequence, 10), extname, path)
	return ioutil.ReadFile(full)
}
func (f *FileExtRW) Put(ctx context.Context, extname string, busVersion bus.Version, path string, content []byte) error {
	path = filepath.Clean(path)
	full := filepath.Join(f.root, "ext", strconv.FormatInt(busVersion.Sequence, 10), extname, path)
	return ioutil.WriteFile(full, content, 0600)
}
