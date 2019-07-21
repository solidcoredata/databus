package bus

// DeltaBus needs to handle the following changes:
//  * New Node
//  * Remove Node
//  * Node name changes
//  * Update Node Role Field
//    - Add Role Field
//    - Remove Role Field
//    - Update Role Field
//    - Rename Role Field
type DeltaBus struct {
	Current  *Bus
	Previous *Bus
	Actions  []DeltaAction
}

type DeltaAction struct {
	Alter         Alter
	NodeCurrent   *Node
	NodePrevious  *Node
	FieldCurrent  *Field
	FieldPrevious *Field
}

type Alter int32

const (
	AlterNothing Alter = iota
	AlterNodeAdd
	AlterNodeRemove
	AlterNodeRename
	AlterFieldAdd
	AlterFieldRemove
	AlterFieldRename
	AlterFieldUpdate
)

type DeltaNode struct {
	Node string
}

type DeltaField struct {
	Node  string
	Role  string
	Side  Side
	Field string
}

func NewDelta(current, previous *Bus) (*DeltaBus, error) {
	db := &DeltaBus{
		Current:  current,
		Previous: previous,
	}
	// TODO(daniel.theophanes): Calculate node and field level actions.
	return db, nil
}

func (*DeltaBus) String() string {
	panic("TODO")
}
