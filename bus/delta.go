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

type DeltaAction struct{}

type Alter int32

const (
	AlterNothing Alter = iota
	AlterAdd
	AlterRemove
	AlterRename
	AlterUpdate // Only used for in DeltaField.
)

type DeltaNode struct {
	Node         string
	PreviousName string
}

type DeltaField struct {
	Node         string
	PreviousName string
	Role         string
	Side         Side
	Field        string
}

func NewDelta(current, previous *Bus) (*DeltaBus, error) {
	db := &DeltaBus{
		Current:  current,
		Previous: previous,
	}
	return db, nil
}

func (b *Bus) NodesTopological() []*Node {
	return nil
}

func (*DeltaBus) String() string {
	panic("TODO")
}
