package bus

type Node struct {
	Name    string   // Name of the node.
	NameAlt []string // Previous name of the node. Useful for renames.
	Type    string
	Roles   []Role
	Binds   []Bind

	names           []string           `json:"-"`
	nodeType        *NodeType          `json:"-"`
	roleLookup      map[string]*Role   `json:"-"`
	bindAliasLookup map[string]*Bind   `json:"-"`
	bindNameLookup  map[string][]*Bind `json:"-"`
}
type Bind struct {
	Alias string
	Name  string

	node *Node `json:"-"`
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

	roleLookup map[string]*RoleType `json:"-"`
}
type Property struct {
	Name     string
	Type     string
	Optional bool
	Send     bool // Set the value in other connected nodes that have Recv == true.
	Recv     bool
	Default  interface{}

	// defaultValue is logically the same as Default, but normalized and typed.
	defaultValue interface{}
}
type RoleType struct {
	Name string
	// TODO(daniel.theophanes): Add FieldCount (One | ZeroPlus | OnePlus).
	// DB table would be OnePlus, property row would be One, optional list would be ZeroPlus.
	Properties []Property

	propNameLookup map[string]*Property `json:"-"`
}
type Role struct {
	Name   string
	Side   Side
	Fields []Field // Each field must match the Node Type role properties.

	fieldIDLookup map[int64]*Field `json:"-"`
	roleType      *RoleType        `json:"-"`
}
type KV = map[string]interface{}
type Field struct {
	// The field ID only needs to be set to a non-zero value before attempting to rename a stateful field.
	// Once set, it should not be changed. ID value should not imply order.
	ID int64

	// Bound Alias name.
	Alias string
	KV    KV

	// Same logical values as in KV, but each value is normalized to
	// the field type, defaults taken into account.
	// If a value for KV is absent, but has a property, then it is
	// entered in values with a value of nil.
	values KV
}

// Value returns the field value taking into account the role type property.
// If name is not a valid property, Value will panic.
func (f *Field) Value(name string) interface{} {
	return nil
}

type Version struct {
	Version int64
}

type Bus struct {
	// Version is only set on checkpoint.
	Version Version

	Nodes []Node
	Types []NodeType

	// setup is true after the lookup fields are setup.
	setup bool

	nodeLookup map[string]*Node     `json:"-"`
	typeLookup map[string]*NodeType `json:"-"`
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
