package bus

import (
	"encoding/base32"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"math"
	"strconv"

	"github.com/cockroachdb/apd/v2"
	"solidcoredata.org/src/databus/internal/tsort"
)

var _ tsort.NodeCollection = (*bussort)(&Bus{})

type bussort Bus

func (bs *bussort) Index(i int) tsort.Node {
	return tsort.Node(bs.Nodes[i])
}
func (bs *bussort) Len() int {
	return len(bs.Nodes)
}
func (bs *bussort) Swap(i, j int) {
	bs.Nodes[i], bs.Nodes[j] = bs.Nodes[j], bs.Nodes[i]
}

func (b *Bus) findNode(name string) *Node {
	return b.nodeLookup[name]
}

// Init should populate lookup fields, as well as return
// any basic errors such as duplicate names.
//
// TODO(daniel.theophanes): Add an option to ignore missing node ref names.
// Will be useful when init a partial bus sent to runners.
func (b *Bus) Init() error {
	var errs *Errors

	b.setup = false
	b.nodeLookup = make(map[string]*Node, len(b.Nodes))
	b.typeLookup = make(map[string]*NodeType, len(b.Types))
	b.nodeByType = make(map[string][]*Node, len(b.Types))

	for ni := range b.Types {
		nt := &b.Types[ni]
		nt.roleLookup = make(map[string]*RoleType, len(nt.Roles))

		if _, ok := b.typeLookup[nt.Name]; ok {
			errs = errs.AppendMsg("bus: node type %q already defined", nt.Name)
			continue
		}
		b.typeLookup[nt.Name] = nt
		for ri := range nt.Roles {
			r := &nt.Roles[ri]
			r.propNameLookup = make(map[string]*Property, len(r.Properties))
			if _, ok := nt.roleLookup[r.Name]; ok {
				errs = errs.AppendMsg("bus: node type %q re-defines role %q", nt.Name, r.Name)
				continue
			}
			nt.roleLookup[r.Name] = r

			if len(r.Properties) == 0 {
				errs = errs.AppendMsg("bus: node type %q role %q contains zero properties, at least one is required", nt.Name, r.Name)
			}

			for ri := range r.Properties {
				pr := &r.Properties[ri]
				pr.defaultValue = nil

				if _, ok := r.propNameLookup[pr.Name]; ok {
					errs = errs.AppendMsg("bus: node type %q role %q re-defines property %q", nt.Name, r.Name, pr.Name)
					continue
				}
				if !validType(pr.Type) {
					errs = errs.AppendMsg("bus: node type %q role %q property %q, invalid type %q", nt.Name, r.Name, pr.Name, pr.Type)
					continue
				}
				// Validate property default.
				if pr.Default != nil {
					if value, err := validValue(pr.Type, pr.Default, b.findNode); err != nil {
						errs = errs.AppendMsg("bus: node type %q role %q property %q invalid default for %v: %v", nt.Name, r.Name, pr.Name, pr.Default, err)
						continue
					} else {
						pr.defaultValue = value
					}
				}
				r.propNameLookup[pr.Name] = pr
			}
		}
	}
	for ni := range b.Nodes {
		n := &b.Nodes[ni]
		n.roleLookup = make(map[string]*Role, len(n.Roles))
		n.bindAliasLookup = make(map[string]*Bind, len(n.Binds))
		n.bindNameLookup = make(map[string][]*Bind, len(n.Binds))
		n.nodeType = nil

		if nt, ok := b.typeLookup[n.Type]; ok {
			n.nodeType = nt
		} else {
			errs = errs.AppendMsg("bus: node %q missing node type %q", n.Name, n.Type)
			continue
		}
		if _, ok := b.nodeLookup[n.Name]; ok {
			errs = errs.AppendMsg("bus: node %q already defined", n.Name)
			continue
		}
		// Create bind lookups.
		b.nodeLookup[n.Name] = n
		b.nodeByType[n.nodeType.Name] = append(b.nodeByType[n.nodeType.Name], n)

		if len(n.NameAlt) > 0 {
			for _, alt := range n.NameAlt {
				if _, ok := b.nodeLookup[alt]; ok {
					errs = errs.AppendMsg("bus: node %q already defined (alt) %q", n.Name, alt)
					continue
				}
				// Create bind lookups.
				b.nodeLookup[alt] = n
			}
		}
	}

	for ni := range b.Nodes {
		n := &b.Nodes[ni]

		if n.nodeType == nil {
			continue
		}
		nt := n.nodeType

		for bi := range n.Binds {
			bd := &n.Binds[bi]
			bd.node = nil

			if len(bd.Alias) == 0 {
				errs = errs.AppendMsg("bus: node %q bind index %d %q missing alias", n.Name, bi, bd.Name)
				continue
			}
			if _, ok := n.bindAliasLookup[bd.Alias]; ok {
				errs = errs.AppendMsg("bus: node %q already bound alias %q", n.Name, bd.Alias)
				continue
			}
			if boundNode, ok := b.nodeLookup[bd.Name]; ok {
				bd.node = boundNode
			} else {
				errs = errs.AppendMsg("bus: node %q bind alias %q invalid node name %q", n.Name, bd.Alias, bd.Name)
				continue
			}

			n.bindAliasLookup[bd.Alias] = bd
			n.bindNameLookup[bd.Name] = append(n.bindNameLookup[bd.Name], bd)
		}
		for ri := range n.Roles {
			r := &n.Roles[ri]
			r.fieldIDLookup = make(map[int64]*Field, len(r.Fields))
			r.roleType = nil

			if rt, ok := nt.roleLookup[r.Name]; ok {
				r.roleType = rt
			} else {
				errs = errs.AppendMsg("bus: node %q role type %q not found", n.Name, r.Name)
				continue
			}
			if _, ok := n.roleLookup[r.Name]; ok {
				errs = errs.AppendMsg("bus: node %q re-defines role %q", n.Name, r.Name)
				continue
			}
			n.roleLookup[r.Name] = r

			// Check field count property.
			switch r.roleType.FieldCount {
			default:
				errs = errs.AppendMsg("bus: unknown FieldCount in role type %q: %v", r.roleType.Name, r.roleType.FieldCount)
			case ZeroPlus:
				// All lengths of Fields okay.
			case One:
				if len(r.Fields) != 1 {
					errs = errs.AppendMsg("bus: node %q role %q expects one field, but has %d", n.Name, r.Name, len(r.Fields))
				}
			case OnePlus:
				if len(r.Fields) == 0 {
					errs = errs.AppendMsg("bus: node %q role %q expects one or more fields, but has zero", n.Name, r.Name)
				}
			}

			// Verify fields and aliases.
			for fi := range r.Fields {
				f := &r.Fields[fi]
				f.values = make(KV, len(r.roleType.Properties))

				if len(f.Alias) > 0 {
					if _, ok := n.bindAliasLookup[f.Alias]; !ok {
						errs = errs.AppendMsg("bus: node %q role %q field index %d invlid bind alias %q", n.Name, r.Name, fi, f.Alias)
						continue
					}
				}
				for key, value := range f.KV {
					pr, ok := r.roleType.propNameLookup[key]
					if !ok {
						errs = errs.AppendMsg("bus: node %q role %q field index %d invalid key %q", n.Name, r.Name, fi, key)
						continue
					}
					// Validate node values.
					if value, err := validValue(pr.Type, value, b.findNode); err != nil {
						errs = errs.AppendMsg("bus: node %q role %q field index %d invalid value for type %q: %v", n.Name, r.Name, fi, key, err)
						continue
					} else {
						f.values[key] = value
					}
				}
				for _, pr := range r.roleType.propNameLookup {
					_, found := f.values[pr.Name]
					if found {
						continue
					}
					f.values[pr.Name] = pr.defaultValue
				}
			}
		}
		// Verify Node has all Roles in Role Type.
		for name := range nt.roleLookup {
			if _, ok := n.roleLookup[name]; !ok {
				errs = errs.AppendMsg("bus: node %q missing role %q as defined in role type %q", n.Name, name, n.Type)
				continue
			}
		}
	}
	if errs == nil {
		b.setup = true
		return nil
	}
	err := tsort.Sort((*bussort)(b))
	if err != nil {
		errs = errs.Append(err)
	}
	return errs
}

// validType checks that the type name is a valid type.
// Keep in sync with validValue.
func validType(tp string) bool {
	switch tp {
	default:
		return false
	case "text": // string
	case "int": // int64
	case "float": // float64
	case "bool": // bool
	case "decimal": // *apd.Decimal
	case "bytes": // []byte
	case "node": // *Node
	}
	return true
}

// validValue takes the type name and value, and verifies it is a valid type and returns
// a normalized value. For example a type with "int" and a string value of "123" will return
// an int64 value of 123.
// Keep in sync with validType.
func validValue(tp string, v interface{}, findNode func(name string) *Node) (interface{}, error) {
	switch tp {
	default:
		return nil, fmt.Errorf("unknown type %s", tp)
	case "text":
		s, ok := v.(string)
		if !ok {
			return nil, fmt.Errorf("expected %[1]s got %[2]T (%[2]v)", tp, v)
		}
		return s, nil
	case "int":
		switch v := v.(type) {
		default:
			return nil, fmt.Errorf("expected %[1]s got %[2]T (%[2]v)", tp, v)
		case json.Number:
			n, err := strconv.ParseInt(string(v), 10, 64)
			if err != nil {
				return nil, fmt.Errorf("expected %[1]s got %[2]v: %v", tp, v, err)
			}
			return n, nil
		case string:
			n, err := strconv.ParseInt(v, 10, 64)
			if err != nil {
				return nil, fmt.Errorf("expected %[1]s got %[2]v: %v", tp, v, err)
			}
			return n, nil
		case int:
			return int64(v), nil
		case int64:
			return v, nil
		case float32:
			x := float64(v)
			if math.Floor(x) != x {
				return nil, fmt.Errorf("an %s may not have a decimal part in %v", tp, v)
			}
			return int64(v), nil
		case float64:
			if math.Floor(v) != v {
				return nil, fmt.Errorf("%s may not have a decimal part in %v", tp, v)
			}
			return int64(v), nil
		}
	case "float":
		switch v := v.(type) {
		default:
			return nil, fmt.Errorf("expected %[1]s got %[2]T (%[2]v)", tp, v)
		case json.Number:
			f, err := strconv.ParseFloat(string(v), 64)
			if err != nil {
				return nil, fmt.Errorf("expected %[1]s got %[2]v: %v", tp, v, err)
			}
			return f, nil
		case string:
			f, err := strconv.ParseFloat(v, 64)
			if err != nil {
				return nil, fmt.Errorf("expected %[1]s got %[2]v: %v", tp, v, err)
			}
			return f, nil
		case int:
			f := float64(v)
			i2 := int(f)
			if v != i2 {
				return nil, fmt.Errorf("%s cannot represent %v", tp, v)
			}
			return f, nil
		case int32:
			f := float64(v)
			i2 := int32(f)
			if v != i2 {
				return nil, fmt.Errorf("%s cannot represent %v", tp, v)
			}
			return f, nil
		case int64:
			f := float64(v)
			i2 := int64(f)
			if v != i2 {
				return nil, fmt.Errorf("%s cannot represent %v", tp, v)
			}
			return f, nil
		case float32:
			return float64(v), nil
		case float64:
			return v, nil
		}
	case "bool":
		switch v := v.(type) {
		default:
			return nil, fmt.Errorf("expected %[1]s got %[2]T (%[2]v)", tp, v)
		case string:
			val, err := strconv.ParseBool(v)
			if err != nil {
				return nil, err
			}
			return val, nil
		case json.Number:
			val, err := strconv.ParseBool(string(v))
			if err != nil {
				return nil, err
			}
			return val, nil
		case bool:
			return v, nil
		}
	case "decimal":
		switch v := v.(type) {
		default:
			return nil, fmt.Errorf("expected %[1]s got %[2]T (%[2]v)", tp, v)
		case string:
			dec, _, err := apd.NewFromString(v)
			if err != nil {
				return nil, err
			}
			return dec, nil
		case json.Number:
			dec, _, err := apd.NewFromString(string(v))
			if err != nil {
				return nil, err
			}
			return dec, nil
		}
	case "bytes":
		switch v := v.(type) {
		default:
			return nil, fmt.Errorf("expected %[1]s got %[2]T (%[2]v)", tp, v)
		case string:
			if len(v) == 0 {
				return []byte{}, nil
			}
			if len(v) < 3 {
				return nil, fmt.Errorf("missing prefix for %s, must be one of 16x, 32x, 64x", tp)
			}
			prefix := v[:3]
			bytea := v[3:]
			switch prefix {
			default:
				return nil, fmt.Errorf("unknown prefix %q, must be one of 16x, 32x, 64x", prefix)
			case "16x":
				bb, err := hex.DecodeString(bytea)
				return bb, err
			case "32x":
				bb, err := base32.StdEncoding.DecodeString(bytea)
				return bb, err
			case "64x":
				bb, err := base64.StdEncoding.DecodeString(bytea)
				return bb, err
			}
		}
	case "node":
		s, ok := v.(string)
		if !ok {
			return nil, fmt.Errorf("%s must be a node name", tp)
		}
		nd := findNode(s)
		if nd == nil {
			return nil, fmt.Errorf("%s %s is not a valid node name", tp, v)
		}
		return nd, nil
	}
}
