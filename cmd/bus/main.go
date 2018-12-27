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

type Errors struct {
	Next *Errors
	Err  error
}

func (errs *Errors) Append(err error) *Errors {
	if errs == nil {
		return &Errors{Err: err}
	}
	if errs.Next == nil {
		errs.Next = &Errors{Err: err}
		return errs
	}
	errs.Next.Append(err)
	return errs
}
func (errs *Errors) AppendMsg(f string, v ...interface{}) *Errors {
	err := fmt.Errorf(f, v...)
	return errs.Append(err)
}
func (errs *Errors) Error() string {
	if errs == nil {
		return ""
	}
	b := &strings.Builder{}
	errs.writeTo(b)
	return b.String()
}
func (errs *Errors) writeTo(b *strings.Builder) {
	if errs == nil {
		return
	}
	if errs.Err != nil {
		b.WriteString(errs.Err.Error())
		b.WriteRune('\n')
	}
	if errs.Next != nil {
		errs.Next.writeTo(b)
	}
}

type program struct{}

// TODO(daniel.theophanes): determine valid types and values.
// Need to ensure valid values can check valid node names.
func (p *program) validType(tp string) bool {
	return true
}
func (p *program) validValue(tp string, v interface{}) error {
	return nil
}

// validate looks for the root definition, loads it,
// then validates it for basic correctness.
func (p *program) validate(ctx context.Context, busPath string) error {
	bus, err := p.loadBus(ctx, busPath)
	if err != nil {
		return err
	}
	var errs *Errors
	type lookupRoleType struct {
		RoleType   *RoleType
		Properties map[string]*Property
	}
	type lookupRole struct {
		lookupRoleType
		Role *Role
	}
	type lookupNodeType struct {
		NodeType  *NodeType
		RoleTypes map[string]lookupRoleType
	}
	type lookupNode struct {
		lookupNodeType
		Node  *Node
		Roles map[string]lookupRole
		Binds map[string]*Bind // Key is alias.
	}
	type lookupBus struct {
		Types map[string]lookupNodeType
		Nodes map[string]lookupNode
	}
	lb := lookupBus{
		Types: make(map[string]lookupNodeType, len(bus.Types)),
		Nodes: make(map[string]lookupNode, len(bus.Nodes)),
	}
	for ni := range bus.Types {
		nt := &bus.Types[ni]
		lnt := lookupNodeType{
			NodeType:  nt,
			RoleTypes: make(map[string]lookupRoleType, len(nt.Roles)),
		}
		if _, ok := lb.Types[nt.Name]; ok {
			errs = errs.AppendMsg("bus: node type %q already defined", nt.Name)
			continue
		}
		lb.Types[nt.Name] = lnt
		for ri := range nt.Roles {
			r := &nt.Roles[ri]
			lrt := lookupRoleType{
				RoleType:   r,
				Properties: make(map[string]*Property, len(r.Properties)),
			}
			if _, ok := lnt.RoleTypes[r.Name]; ok {
				errs = errs.AppendMsg("bus: node type %q re-defines role %q", nt.Name, r.Name)
				continue
			}
			lnt.RoleTypes[r.Name] = lrt
			for ri := range r.Properties {
				pr := &r.Properties[ri]
				if _, ok := lrt.Properties[pr.Name]; ok {
					errs = errs.AppendMsg("bus: node type %q role %q re-defines property %q", nt.Name, r.Name, pr.Name)
					continue
				}
				if !p.validType(pr.Type) {
					errs = errs.AppendMsg("bus: node type %q role %q property %q, invalid type %q", nt.Name, r.Name, pr.Name, pr.Type)
					continue
				}
				lrt.Properties[pr.Name] = pr
			}
		}
	}
	for ni := range bus.Nodes {
		n := &bus.Nodes[ni]
		ln := lookupNode{
			Node:  n,
			Roles: make(map[string]lookupRole, len(n.Roles)),
			Binds: make(map[string]*Bind, len(n.Binds)),
		}
		if lnt, ok := lb.Types[n.Type]; ok {
			ln.lookupNodeType = lnt
		} else {
			errs = errs.AppendMsg("bus: node %q missing node type %q", n.Name, n.Type)
			continue
		}
		if _, ok := lb.Nodes[n.Name]; ok {
			errs = errs.AppendMsg("bus: node %q already defined", n.Name)
			continue
		}
		// Create bind lookups.
		lb.Nodes[n.Name] = ln
		for bi := range n.Binds {
			b := &n.Binds[bi]
			if len(b.Alias) == 0 {
				errs = errs.AppendMsg("bus: node %q bind index %d %q missing alias", n.Name, bi, b.Name)
				continue
			}
			if _, ok := ln.Binds[b.Alias]; ok {
				errs = errs.AppendMsg("bus: node %q already bound alias %q", n.Name, b.Alias)
				continue
			}
			ln.Binds[b.Alias] = b
		}
		for ri := range n.Roles {
			r := &n.Roles[ri]
			lr := lookupRole{
				Role: r,
			}
			if lrt, ok := ln.RoleTypes[r.Name]; ok {
				lr.lookupRoleType = lrt
			} else {
				errs = errs.AppendMsg("bus: node %q role type %q not found", n.Name, r.Name)
				continue
			}
			if _, ok := ln.Roles[r.Name]; ok {
				errs = errs.AppendMsg("bus: node %q re-defines role %q", n.Name, r.Name)
				continue
			}
			ln.Roles[r.Name] = lr
			// Verify fields and aliases.
			for fi, f := range r.Fields {
				if len(f.Alias) > 0 {
					if _, ok := ln.Binds[f.Alias]; !ok {
						errs = errs.AppendMsg("bus: node %q role %q field index %d invlid bind alias %q", n.Name, r.Name, fi, f.Alias)
						continue
					}
				}
				for key, value := range f.KV {
					pr, ok := lr.Properties[key]
					if !ok {
						errs = errs.AppendMsg("bus: node %q role %q field index %d invalid key %q", n.Name, r.Name, fi, key)
						continue
					}
					// TODO(daniel.theophanes): validate node values.
					if err := p.validValue(pr.Type, value); err != nil {
						errs = errs.AppendMsg("bus: node %q role %q field index %d invalid value for type %q: %v", n.Name, r.Name, fi, key, err)
						continue
					}
				}
			}
		}
		// Verify Node has all Roles in Role Type.
		for name := range ln.RoleTypes {
			if _, ok := ln.Roles[name]; !ok {
				errs = errs.AppendMsg("bus: node %q missing role %q as defined in role type %q", n.Name, name, n.Type)
				continue
			}
		}
	}
	// Loop through nodes again and verify bind names.
	for _, ln := range lb.Nodes {
		for _, b := range ln.Binds {
			if _, ok := lb.Nodes[b.Name]; !ok {
				errs = errs.AppendMsg("bus: node %q bind alias %q invalid node name %q", ln.Node.Name, b.Alias, b.Name)
				continue
			}
		}
	}
	return errs
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
		coder.UseNumber()
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
		coder.UseNumber()
		err = coder.Decode(bus)
		if err != nil {
			return nil, fmt.Errorf("bus: unable to unmarshal %q: %v", busPath, err)
		}
		return bus, nil
	}
	return nil, fmt.Errorf("bus: unknown file extention %q", ext)
}

type Node struct {
	Name  string
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
	Name  string
	Roles []RoleType
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
	Alias string
	KV    KV
}
type Bus struct {
	Nodes []Node
	Types []NodeType
}
