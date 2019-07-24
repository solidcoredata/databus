package bus

import (
	"fmt"
)

// Filter bus by node types.
func (b *Bus) Filter(types []string) *Bus {
	tlookup := make(map[string]bool, len(types))
	for _, t := range types {
		tlookup[t] = true
	}

	f := &Bus{
		Version: b.Version,
		Nodes:   make([]Node, 0, len(b.Nodes)),
		Types:   make([]NodeType, 0, len(types)),
	}
	for _, n := range b.Nodes {
		if !tlookup[n.Type] {
			continue
		}
		f.Nodes = append(f.Nodes, n)
	}
	for _, t := range b.Types {
		if !tlookup[t.Name] {
			continue
		}
		f.Types = append(f.Types, t)
	}
	return f
}

// Node returns a *Node by name.
func (b *Bus) Node(name string) *Node {
	if b == nil {
		return nil
	}
	return b.nodeLookup[name]
}

func (b *Bus) NodeByType(typeName ...string) []*Node {
	if b == nil {
		return nil
	}
	ret := make([]*Node, 0, 20)
	for _, tn := range typeName {
		list := b.nodeByType[tn]
		if len(list) == 0 {
			continue
		}
		ret = append(ret, list...)
	}
	return ret
}

// NodeType returns a *NodeType by node type name.
func (b *Bus) NodeType(name string) *NodeType {
	return b.typeLookup[name]
}

// RoleType returns the RoleType name.
func (nt *NodeType) RoleType(name string) *RoleType {
	return nt.roleLookup[name]
}

// Property returns the Property name.
func (rt *RoleType) Property(name string) *Property {
	return rt.propNameLookup[name]
}

// Return the normalized default value.
func (p *Property) DefaultValue() interface{} {
	return p.defaultValue
}

// NodeType returns the associated NodeType to the Node.
func (n *Node) NodeType() *NodeType {
	return n.nodeType
}

// Role returns the Role name.
func (n *Node) Role(name string) *Role {
	return n.roleLookup[name]
}

// RoleType returns the associated RoleType.
func (r *Role) RoleType() *RoleType {
	return r.roleType
}

// BindAlias returns the Bind by alias.
func (n *Node) BindAlias(alias string) *Bind {
	return n.bindAliasLookup[alias]
}

// Value returns the field value taking into account the role type property.
// If name is not a valid property, Value will panic.
func (f *Field) Value(name string) interface{} {
	v, found := f.values[name]
	if !found {
		panic(fmt.Errorf("property %q not present", name))
	}
	return v
}

func (f *Field) Name() string {
	return f.name
}

func (f *Field) needUpdate(prev *Field) bool {
	// TODO(daniel.theophanes): This is probably overly simple, but may work for now.
	for fkey, fvalue := range f.values {
		if prev.values[fkey] != fvalue {
			return true
		}
	}
	return false
}

// Node returns the associated Node to the Bind.
func (b *Bind) Node() *Node {
	return b.node
}
