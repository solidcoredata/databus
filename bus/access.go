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

// Value returns the field value taking into account the role type property.
// If name is not a valid property, Value will panic.
func (f *Field) Value(name string) interface{} {
	v, found := f.values[name]
	if !found {
		panic(fmt.Errorf("property %q not present", name))
	}
	return v
}
