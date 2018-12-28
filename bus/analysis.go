package bus

import (
	"context"
	"encoding/base32"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"math"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/cockroachdb/apd"
	"github.com/google/go-jsonnet"
)

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

type Analysis struct {
	lookupBus
}

// validType verifies the type names are valid.
func (a *Analysis) validType(tp string) bool {
	switch tp {
	default:
		return false
	case "text":
	case "int":
	case "float":
	case "decimal":
	case "bytea":
	case "node":
	}
	return true
}
func (a *Analysis) validValue(tp string, v interface{}) error {
	switch tp {
	default:
		return fmt.Errorf("unknown type %s", tp)
	case "text":
		_, ok := v.(string)
		if !ok {
			return fmt.Errorf("expected %[1]s got %[2]T (%[2]v)", tp, v)
		}
		return nil
	case "int":
		switch v := v.(type) {
		default:
			return fmt.Errorf("expected %[1]s got %[2]T (%[2]v)", tp, v)
		case json.Number:
			_, err := strconv.ParseInt(string(v), 10, 64)
			if err != nil {
				return fmt.Errorf("expected %[1]s got %[2]v: %v", tp, v, err)
			}
			return nil
		case string:
			_, err := strconv.ParseInt(v, 10, 64)
			if err != nil {
				return fmt.Errorf("expected %[1]s got %[2]v: %v", tp, v, err)
			}
			return nil
		case int:
			return nil
		case int64:
			return nil
		case float32:
			x := float64(v)
			if math.Floor(x) != x {
				return fmt.Errorf("an %s may not have a decimal part in %v", tp, v)
			}
			return nil
		case float64:
			if math.Floor(v) != v {
				return fmt.Errorf("%s may not have a decimal part in %v", tp, v)
			}
			return nil
		}
	case "float":
		switch v := v.(type) {
		default:
			return fmt.Errorf("expected %[1]s got %[2]T (%[2]v)", tp, v)
		case json.Number:
			_, err := strconv.ParseFloat(string(v), 64)
			if err != nil {
				return fmt.Errorf("expected %[1]s got %[2]v: %v", tp, v, err)
			}
			return nil
		case string:
			_, err := strconv.ParseFloat(v, 64)
			if err != nil {
				return fmt.Errorf("expected %[1]s got %[2]v: %v", tp, v, err)
			}
			return nil
		case int:
			f := float64(v)
			i2 := int(f)
			if v != i2 {
				return fmt.Errorf("%s cannot represent %v", tp, v)
			}
			return nil
		case int32:
			f := float64(v)
			i2 := int32(f)
			if v != i2 {
				return fmt.Errorf("%s cannot represent %v", tp, v)
			}
			return nil
		case int64:
			f := float64(v)
			i2 := int64(f)
			if v != i2 {
				return fmt.Errorf("%s cannot represent %v", tp, v)
			}
			return nil
		case float32:
			return nil
		case float64:
			return nil
		}
	case "decimal":
		switch v := v.(type) {
		default:
			return fmt.Errorf("expected %[1]s got %[2]T (%[2]v)", tp, v)
		case string:
			_, _, err := apd.NewFromString(v)
			if err != nil {
				return err
			}
			return nil
		case json.Number:
			_, _, err := apd.NewFromString(string(v))
			if err != nil {
				return err
			}
			return nil
		}
	case "bytea":
		switch v := v.(type) {
		default:
			return fmt.Errorf("expected %[1]s got %[2]T (%[2]v)", tp, v)
		case string:
			if len(v) == 0 {
				return nil
			}
			if len(v) < 3 {
				return fmt.Errorf("missing prefix for %s, must be one of 16x, 32x, 64x", tp)
			}
			prefix := v[:3]
			bytea := v[3:]
			switch prefix {
			default:
				return fmt.Errorf("unknown prefix %q, must be one of 16x, 32x, 64x", prefix)
			case "16x":
				_, err := hex.DecodeString(bytea)
				return err
			case "32x":
				_, err := base32.StdEncoding.DecodeString(bytea)
				return err
			case "64x":
				_, err := base64.StdEncoding.DecodeString(bytea)
				return err
			}
		}
	case "node":
		s, ok := v.(string)
		if !ok {
			return fmt.Errorf("%s must be a node name", tp)
		}
		if _, ok := a.Nodes[s]; !ok {
			return fmt.Errorf("%s %s is not a valid node name", tp, v)
		}
		return nil
	}
}

// validate looks for the root definition, loads it,
// then validates it for basic correctness.
func (a *Analysis) Validate(ctx context.Context, bus *Bus) error {
	var errs *Errors
	lb := lookupBus{
		Types: make(map[string]lookupNodeType, len(bus.Types)),
		Nodes: make(map[string]lookupNode, len(bus.Nodes)),
	}
	a.lookupBus = lb
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
				if !a.validType(pr.Type) {
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
	}

	for ni := range bus.Nodes {
		n := &bus.Nodes[ni]
		ln := lb.Nodes[n.Name]

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
			if _, ok := lb.Nodes[b.Name]; !ok {
				errs = errs.AppendMsg("bus: node %q bind alias %q invalid node name %q", ln.Node.Name, b.Alias, b.Name)
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
					// Validate node values.
					if err := a.validValue(pr.Type, value); err != nil {
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
	if errs == nil {
		return nil
	}
	return errs
}
func LoadBus(ctx context.Context, busPath string) (*Bus, error) {
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

		bus, err := loadBusReader(ctx, f)
		if err != nil {
			return nil, fmt.Errorf("bus: for %q %v", busPath, err)
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
		bus, err := loadBusReader(ctx, strings.NewReader(out))
		if err != nil {
			return nil, fmt.Errorf("bus: for %q %v", busPath, err)
		}
		return bus, nil
	}
	return nil, fmt.Errorf("bus: unknown file extention %q", ext)
}
func LoadBusReader(ctx context.Context, r io.Reader) (*Bus, error) {
	bus, err := loadBusReader(ctx, r)
	if err != nil {
		return nil, fmt.Errorf("bus: %v", err)
	}
	return bus, nil
}
func loadBusReader(ctx context.Context, r io.Reader) (*Bus, error) {
	bus := &Bus{}
	coder := json.NewDecoder(r)
	coder.DisallowUnknownFields()
	coder.UseNumber()
	err := coder.Decode(bus)
	if err != nil {
		return nil, fmt.Errorf("unable to unmarshal: %v", err)
	}
	return bus, nil
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