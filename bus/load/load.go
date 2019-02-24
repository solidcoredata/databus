package load

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"

	"solidcoredata.org/src/databus/bus"

	"github.com/google/go-jsonnet"
)

func Decode(ctx context.Context, p string, v interface{}) error {
	ext := filepath.Ext(p)
	switch ext {
	default:
		return fmt.Errorf("bus/load: unknown file ext %q", ext)
	case ".json":
		f, err := os.Open(p)
		if err != nil {
			return fmt.Errorf("bus/load: unable to open file %q: %v", p, err)
		}
		defer f.Close()

		err = decodeReader(ctx, f, v)
		if err != nil {
			return fmt.Errorf("bus/load: for %q %v", p, err)
		}
		return nil
	case ".jsonnet":
		vm := jsonnet.MakeVM()
		dir, _ := filepath.Split(p)
		vm.Importer(&jsonnet.FileImporter{
			JPaths: []string{dir},
		})
		bb, err := ioutil.ReadFile(p)
		if err != nil {
			return fmt.Errorf("bus/load: unable to open file %q: %v", p, err)
		}
		out, err := vm.EvaluateSnippet(p, string(bb))
		if err != nil {
			return fmt.Errorf("bus/load: %v", err)
		}
		err = decodeReader(ctx, strings.NewReader(out), v)
		if err != nil {
			return fmt.Errorf("bus/load: for %q %v", p, err)
		}
		return nil
	}
}

func DecodeReader(ctx context.Context, r io.Reader, v interface{}) error {
	err := decodeReader(ctx, r, v)
	if err != nil {
		return fmt.Errorf("bus/load: %v", err)
	}
	return nil
}

func decodeReader(ctx context.Context, r io.Reader, v interface{}) error {
	coder := json.NewDecoder(r)
	coder.DisallowUnknownFields()
	coder.UseNumber()
	err := coder.Decode(v)
	if err != nil {
		return fmt.Errorf("unable to unmarshal: %v", err)
	}
	return nil
}

func Bus(ctx context.Context, busPath string) (*bus.Bus, error) {
	bus := &bus.Bus{}
	err := Decode(ctx, busPath, bus)
	return bus, err
}

func BusReader(ctx context.Context, r io.Reader) (*bus.Bus, error) {
	bus := &bus.Bus{}
	err := DecodeReader(ctx, r, bus)
	return bus, err
}
