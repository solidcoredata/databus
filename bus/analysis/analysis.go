package analysis

import (
	"context"
	"encoding/base32"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"math"
	"strconv"

	"solidcoredata.org/src/databus/bus"

	"github.com/cockroachdb/apd"
)

type lookupRoleType struct {
	RoleType   *bus.RoleType
	Properties map[string]*bus.Property
}
type lookupRole struct {
	lookupRoleType
	Role *bus.Role
}
type lookupNodeType struct {
	NodeType  *bus.NodeType
	RoleTypes map[string]lookupRoleType
}
type lookupNode struct {
	lookupNodeType
	Node  *bus.Node
	Roles map[string]lookupRole
	Binds map[string]*bus.Bind // Key is alias.
	Prev  bool
}
type lookupBus struct {
	Types map[string]lookupNodeType
	Nodes map[string]lookupNode
}

type Analysis struct {
	version bus.Version
	lookupBus

	validated bool
}

// validType verifies the type names are valid.
func (a *Analysis) validType(tp string) bool {
	switch tp {
	default:
		return false
	case "text":
	case "int":
	case "float":
	case "bool":
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
	case "bool":
		switch v := v.(type) {
		default:
			return fmt.Errorf("expected %[1]s got %[2]T (%[2]v)", tp, v)
		case string:
			_, err := strconv.ParseBool(v)
			if err != nil {
				return err
			}
			return nil
		case json.Number:
			_, err := strconv.ParseBool(string(v))
			if err != nil {
				return err
			}
			return nil
		case bool:
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

// New creates a new analysis validates a Bus
// to get it ready for further analysis.
//
// validate looks for the root definition, loads it,
// then validates it for basic correctness.
func New(ctx context.Context, b *bus.Bus) (*Analysis, error) {
	// TODO(daniel.theophanes): Also ensure NamePrev (Node|Field) do not conflict with any current names.
	// This is to ensure systems can use this information to still service -1 version clients and not conflict right away.
	// It forces designers to take a two part change to many renames.

	a := &Analysis{
		version: b.Version,
	}
	var errs *bus.Errors
	lb := lookupBus{
		Types: make(map[string]lookupNodeType, len(b.Types)),
		Nodes: make(map[string]lookupNode, len(b.Nodes)),
	}
	a.lookupBus = lb
	for ni := range b.Types {
		nt := &b.Types[ni]
		lnt := lookupNodeType{
			NodeType:  nt,
			RoleTypes: make(map[string]lookupRoleType, len(nt.Roles)),
		}
		if _, ok := lb.Types[nt.Name]; ok {
			errs = errs.AppendMsg("bus/anlysis: node type %q already defined", nt.Name)
			continue
		}
		lb.Types[nt.Name] = lnt
		for ri := range nt.Roles {
			r := &nt.Roles[ri]
			lrt := lookupRoleType{
				RoleType:   r,
				Properties: make(map[string]*bus.Property, len(r.Properties)),
			}
			if _, ok := lnt.RoleTypes[r.Name]; ok {
				errs = errs.AppendMsg("bus/anlysis: node type %q re-defines role %q", nt.Name, r.Name)
				continue
			}
			lnt.RoleTypes[r.Name] = lrt
			for ri := range r.Properties {
				pr := &r.Properties[ri]
				if _, ok := lrt.Properties[pr.Name]; ok {
					errs = errs.AppendMsg("bus/anlysis: node type %q role %q re-defines property %q", nt.Name, r.Name, pr.Name)
					continue
				}
				if !a.validType(pr.Type) {
					errs = errs.AppendMsg("bus/anlysis: node type %q role %q property %q, invalid type %q", nt.Name, r.Name, pr.Name, pr.Type)
					continue
				}
				lrt.Properties[pr.Name] = pr
			}
		}
	}
	for ni := range b.Nodes {
		n := &b.Nodes[ni]
		ln := lookupNode{
			Node:  n,
			Roles: make(map[string]lookupRole, len(n.Roles)),
			Binds: make(map[string]*bus.Bind, len(n.Binds)),
		}
		if lnt, ok := lb.Types[n.Type]; ok {
			ln.lookupNodeType = lnt
		} else {
			errs = errs.AppendMsg("bus/anlysis: node %q missing node type %q", n.Name, n.Type)
			continue
		}
		if _, ok := lb.Nodes[n.Name]; ok {
			errs = errs.AppendMsg("bus/anlysis: node %q already defined", n.Name)
			continue
		}
		// Create bind lookups.
		lb.Nodes[n.Name] = ln

		if len(n.NamePrev) > 0 && n.NamePrev != n.Name {
			ln.Prev = true

			if _, ok := lb.Nodes[n.NamePrev]; ok {
				errs = errs.AppendMsg("bus/anlysis: node %q already defined (prev)", n.NamePrev)
				continue
			}
			// Create bind lookups.
			lb.Nodes[n.NamePrev] = ln
		}
	}

	for ni := range b.Nodes {
		n := &b.Nodes[ni]
		ln := lb.Nodes[n.Name]

		for bi := range n.Binds {
			b := &n.Binds[bi]
			if len(b.Alias) == 0 {
				errs = errs.AppendMsg("bus/anlysis: node %q bind index %d %q missing alias", n.Name, bi, b.Name)
				continue
			}
			if _, ok := ln.Binds[b.Alias]; ok {
				errs = errs.AppendMsg("bus/anlysis: node %q already bound alias %q", n.Name, b.Alias)
				continue
			}
			if _, ok := lb.Nodes[b.Name]; !ok {
				errs = errs.AppendMsg("bus/anlysis: node %q bind alias %q invalid node name %q", ln.Node.Name, b.Alias, b.Name)
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
				errs = errs.AppendMsg("bus/anlysis: node %q role type %q not found", n.Name, r.Name)
				continue
			}
			if _, ok := ln.Roles[r.Name]; ok {
				errs = errs.AppendMsg("bus/anlysis: node %q re-defines role %q", n.Name, r.Name)
				continue
			}
			ln.Roles[r.Name] = lr
			// Verify fields and aliases.
			for fi, f := range r.Fields {
				if len(f.Alias) > 0 {
					if _, ok := ln.Binds[f.Alias]; !ok {
						errs = errs.AppendMsg("bus/anlysis: node %q role %q field index %d invlid bind alias %q", n.Name, r.Name, fi, f.Alias)
						continue
					}
				}
				for key, value := range f.KV {
					pr, ok := lr.Properties[key]
					if !ok {
						errs = errs.AppendMsg("bus/anlysis: node %q role %q field index %d invalid key %q", n.Name, r.Name, fi, key)
						continue
					}
					// Validate node values.
					if err := a.validValue(pr.Type, value); err != nil {
						errs = errs.AppendMsg("bus/anlysis: node %q role %q field index %d invalid value for type %q: %v", n.Name, r.Name, fi, key, err)
						continue
					}
				}
			}
		}
		// Verify Node has all Roles in Role Type.
		for name := range ln.RoleTypes {
			if _, ok := ln.Roles[name]; !ok {
				errs = errs.AppendMsg("bus/anlysis: node %q missing role %q as defined in role type %q", n.Name, name, n.Type)
				continue
			}
		}
	}
	if errs == nil {
		a.validated = true
		return a, nil
	}
	return nil, errs
}

func NewDelta(current, previous *Analysis) (*bus.DeltaBus, error) {
	delta := &bus.DeltaBus{
		Current:  current.version,
		Previous: previous.version,
	}
	return delta, nil
}
