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
	Name     string // Name of the field.
	NamePrev string // Previous name of the field, useful during renames.

	// Bound Alias name.
	Alias string
	KV    KV
}
type Bus struct {
	Nodes []Node
	Types []NodeType
}
