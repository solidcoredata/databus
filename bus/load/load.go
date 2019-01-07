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

func Bus(ctx context.Context, busPath string) (*bus.Bus, error) {
	ext := filepath.Ext(busPath)
	switch ext {
	default:
		return nil, fmt.Errorf("bus/load: unknown file ext %q", ext)
	case ".json":
		f, err := os.Open(busPath)
		if err != nil {
			return nil, fmt.Errorf("bus/load: unable to open file %q: %v", busPath, err)
		}
		defer f.Close()

		bus, err := loadBusReader(ctx, f)
		if err != nil {
			return nil, fmt.Errorf("bus/load: for %q %v", busPath, err)
		}
		return bus, nil
	case ".jsonnet":
		vm := jsonnet.MakeVM()
		dir, _ := filepath.Split(busPath)
		vm.Importer(&jsonnet.FileImporter{
			JPaths: []string{dir},
		})
		bb, err := ioutil.ReadFile(busPath)
		if err != nil {
			return nil, fmt.Errorf("bus/load: unable to open file %q: %v", busPath, err)
		}
		out, err := vm.EvaluateSnippet(busPath, string(bb))
		if err != nil {
			return nil, fmt.Errorf("bus/load: %v", err)
		}
		bus, err := loadBusReader(ctx, strings.NewReader(out))
		if err != nil {
			return nil, fmt.Errorf("bus/load: for %q %v", busPath, err)
		}
		return bus, nil
	}
	return nil, fmt.Errorf("bus/load: unknown file extention %q", ext)
}
func BusReader(ctx context.Context, r io.Reader) (*bus.Bus, error) {
	bus, err := loadBusReader(ctx, r)
	if err != nil {
		return nil, fmt.Errorf("bus/load: %v", err)
	}
	return bus, nil
}
func loadBusReader(ctx context.Context, r io.Reader) (*bus.Bus, error) {
	bus := &bus.Bus{}
	coder := json.NewDecoder(r)
	coder.DisallowUnknownFields()
	coder.UseNumber()
	err := coder.Decode(bus)
	if err != nil {
		return nil, fmt.Errorf("unable to unmarshal: %v", err)
	}
	return bus, nil
}
