# Data Bus

Let's take a small side track from coding the basic poc.
How should the data bus be modeled / represented?

 * Node
   - Role
     - Field


Not only that, but some Roles are directional.
Also, Some nodes may be "adapters" and have to "sides".
Maybe each Role could specify "side: left|right|both".

A node should not implement any functionality.
Each node should have a type that goes with it.

There must be a named node type view, that projects the bound
roles and fields on a single representation. This is the view that
are consumed by controls.

Lastly, a node's definition should also containe the other bound nodes that
project onto this node. This is used for validation / refactoring /
and projecting properties. Binding a UI to a query should be easy.
I'm not sure how to represent a query binding to N number of tables.

So to do this, each node should have a list of bound nodes and an alias.
Then each field must reference that alias that it binds to. For now assume
Role names match.

```go
type Node struct {
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
    Name string
    Roletypes []RoleType
}
type Property struct {
	Name     string
	Type     string
	Optional bool
}
type RoleType struct {
    Name string
    Properties []Property
}
type Role struct {
    Name string
    Side Side
    Fields []Field // Each field must match the Node Type role properties.
}
type KV = map[string]interface{}
type Field struct {
    // Bound Alias name.
    B  string
    KV KV
}
```

A database table node type may have two roles. One role
list out the columns for the table. The other
includes information such as what database it uses.
That points to another node being the database.
