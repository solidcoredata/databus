package bus

type Node struct {
	Name     string // Name of the node.
	NamePrev string // Previous name of the node. Useful for renames.
	Type     string
	Roles    []Role
	Binds    []Bind
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
	Send     bool // Set the value in other connected nodes that have Recv == true.
	Recv     bool
	Default  string
}
type RoleType struct {
	Name string
	// TODO(daniel.theophanes): Add FieldCount (One | ZeroPlus | OnePlus).
	// DB table would be OnePlus, property row would be One, optional list would be ZeroPlus.
	Properties []Property
}
type Role struct {
	Name   string
	Side   Side
	Fields []Field // Each field must match the Node Type role properties.
}
type KV = map[string]interface{}
type Field struct {
	// The field ID only needs to be set to a non-zero value before attempting to rename a stateful field.
	// Once set, it should not be changed. ID value should not imply order.
	ID int64

	// Bound Alias name.
	Alias string
	KV    KV
}

type Version struct {
	Version int64
}

type Bus struct {
	// Version is only set on checkpoint.
	Version Version

	Nodes []Node
	Types []NodeType
}

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
