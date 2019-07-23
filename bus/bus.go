// Package bus defines and sets up the Bus, Node and Role
// data types.
package bus

// Side each role is on: a "left" side or "right" side.
// The default is to have each role apply to both sides.
type Side byte

const (
	SideBoth Side = iota
	SideLeft
	SideRight
)

// FieldCount is the minimum number of fields that
// should be present for a given role.
type FieldCount byte

const (
	ZeroPlus FieldCount = iota // Optional lists.
	One                        // Property row.
	OnePlus                    // DB table.
)

// Bus is the primary databus definition.
type Bus struct {
	// Version is only set on checkpoint.
	Version Version

	Types []NodeType
	Nodes []Node

	// setup is true after the lookup fields are setup.
	setup bool

	nodeLookup map[string]*Node
	typeLookup map[string]*NodeType
	nodeByType map[string][]*Node // map[NodeType.Name][]*Node
}

// Version of the Bus.
type Version struct {
	Version int64

	// TODO(daniel.theophanes): This should really be a unique hash string and a sequence int64.
	Sequence   int64
	Identifier [64]byte
}

// NodeType defines the types for nodes.
type NodeType struct {
	Name  string
	Roles []RoleType

	roleLookup map[string]*RoleType
}

// RoleType is the role type in a specific NodeType.
// A single RoleType represents a single "table".
type RoleType struct {
	Name string

	Side       Side
	FieldCount FieldCount

	Properties []Property

	propNameLookup map[string]*Property
}

// Property of a RoleType. Each property is an aspect of a single "column".
// To define a node with 5 "columns", where each column has a "name" and a "size",
// Then the RoleType would define two properties: "name" and "size" and the
// Role would define 5 Fields, each with two key value pairs.
type Property struct {
	Name      string
	Type      string
	FieldName bool
	Optional  bool
	Send      bool // Set the value in other connected nodes that have Recv == true.
	Recv      bool
	Default   interface{}

	// defaultValue is logically the same as Default, but normalized and typed.
	defaultValue interface{}
}

// Node is an instance of a NodeType.
type Node struct {
	Name    string   // Name of the node.
	NameAlt []string // Previous name of the node. Useful for renames.
	Type    string
	Roles   []Role
	Binds   []Bind

	names           []string
	nodeType        *NodeType
	roleLookup      map[string]*Role
	bindAliasLookup map[string]*Bind
	bindNameLookup  map[string][]*Bind
}

func (n Node) ID() string {
	return n.Name
}

func (n Node) ToNode() []string {
	ret := make([]string, 0, len(n.Binds))
	for _, b := range n.Binds {
		ret = append(ret, b.node.Name)
	}
	for _, r := range n.Roles {
		for _, f := range r.Fields {
			for _, v := range f.values {
				switch n := v.(type) {
				case *Node:
					ret = append(ret, n.Name)
				}
			}
		}
	}
	return ret
}

// Role of a given Node. Defines a data set for a single rectangular "table"
// where each "cell" will have as many attibutes as the RoleType has properties.
type Role struct {
	Name   string
	Fields []Field // Each field must match the Node Type role properties.

	fieldIDLookup   map[int64]*Field
	fieldNameLookup map[string]*Field
	roleType        *RoleType
}

// Field defines a single "column". Each "column" is made up of one or more properties
// as defined in the RoleType.
type Field struct {
	// The field ID only needs to be set to a non-zero value before attempting to rename a stateful field.
	// Once set, it should not be changed. ID value should not imply order.
	ID int64

	// name of the field to bind to. Set with Property FieldName=true.
	name string

	// Bound Alias name.
	Alias string
	KV    KV

	// Same logical values as in KV, but each value is normalized to
	// the field type, defaults taken into account.
	// If a value for KV is absent, but has a property, then it is
	// entered in values with a value of nil.
	values KV
}

// KV defines a key-value pair, where the key is a string
// and the value is any valid value type.
type KV = map[string]interface{}

// Bind creates an association between an alias name and a Node.
// In a given Node, each alias must be unique, but the Name may
// not be; a given node may bind to the same node in multiple ways.
type Bind struct {
	Alias string
	Name  string

	node *Node
}
