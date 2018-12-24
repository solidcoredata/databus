package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/google/go-jsonnet"
	"github.com/kardianos/task"
)

func main() {
	p := &program{}

	fBus := &task.Flag{Name: "bus", Type: task.FlagString, Default: "bus.jsonnet", Usage: "File name of the bus definition, may be json or jsonnet."}

	cmd := &task.Command{
		Usage: `Solid Core Data Bus

The root of the data bus project is defined by a "X" file.
Tasks are run defined in "Y" file.`,
		Commands: []*task.Command{
			{
				Name:  "validate",
				Usage: "Validate the data bus.",
				Flags: []*task.Flag{fBus},
				Action: task.ActionFunc(func(ctx context.Context, st *task.State, sc task.Script) error {
					busName := st.Default(fBus.Name, "").(string)
					return p.validate(ctx, st.Filepath(busName))
				}),
			},
			{
				Name:  "checkpoint",
				Usage: "Checkpoint the data bus as a new version.",
			},
			{
				Name:  "run",
				Usage: "Run the configured tasks on the data bus.",
			},
		},
	}

	st := task.DefaultState()
	err := cmd.Exec(os.Args[1:]).Run(context.Background(), st, nil)
	if err != nil {
		log.Fatal(err)
	}
}

type program struct{}

// validate looks for the root definition, loads it,
// then validates it for basic correctness.
func (p *program) validate(ctx context.Context, busPath string) error {
	bus, err := p.loadBus(ctx, busPath)
	if err != nil {
		return err
	}
	_ = bus
	return nil
}
func (p *program) loadBus(ctx context.Context, busPath string) (*Bus, error) {
	ext := filepath.Ext(busPath)
	switch ext {
	default:
		return nil, fmt.Errorf("bus: unknown file ext %q", ext)
	case ".json":
		f, err := os.Open(busPath)
		if err != nil {
			return nil, fmt.Errorf("bus: unable to open file %q: %v", busPath, err)
		}
		defer f.Close()

		bus := &Bus{}
		coder := json.NewDecoder(f)
		coder.DisallowUnknownFields()
		err = coder.Decode(bus)
		if err != nil {
			return nil, fmt.Errorf("bus: unable to unmarshal %q: %v", busPath, err)
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
			return nil, fmt.Errorf("bus: unable to open file %q: %v", busPath, err)
		}
		out, err := vm.EvaluateSnippet(busPath, string(bb))
		if err != nil {
			return nil, fmt.Errorf("bus: %v", err)
		}

		bus := &Bus{}
		coder := json.NewDecoder(strings.NewReader(out))
		coder.DisallowUnknownFields()
		err = coder.Decode(bus)
		if err != nil {
			return nil, fmt.Errorf("bus: unable to unmarshal %q: %v", busPath, err)
		}
		return bus, nil
	}
	return nil, fmt.Errorf("bus: unknown file extention %q", ext)
}

type Node struct {
	Type  string
	Roles []Role
	Binds []Bind
}
type Bind struct {
	Alias string
	Name  string
}
type Side int

const (
	SideBoth Side = iota
	SideLeft
	SideRight
)

type NodeType struct {
	Name      string
	RoleTypes []RoleType
}
type Property struct {
	Name     string
	Type     string
	Optional bool
	Send     bool
	Recv     bool
}
type RoleType struct {
	Name       string
	Properties []Property
}
type Role struct {
	Name   string
	Side   Side
	Fields []Field // Each field must match the Node Type role properties.
}
type KV = map[string]interface{}
type Field struct {
	// Bound Alias name.
	B  string
	KV KV
}
type Bus struct {
	Nodes []Node
	Types []NodeType
}
