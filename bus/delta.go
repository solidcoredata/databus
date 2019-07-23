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
	Script        string
}

type Alter int32

const (
	AlterNothing Alter = iota
	AlterScript
	AlterNodeAdd
	AlterNodeRemove
	AlterNodeRename
	AlterFieldAdd
	AlterFieldRemove
	AlterFieldRename
	AlterFieldUpdate
)

func NewDelta(current, previous *Bus) (*DeltaBus, error) {
	var err error
	err = current.Init()
	if err != nil {
		return nil, err
	}
	err = previous.Init()
	if err != nil {
		return nil, err
	}
	db := &DeltaBus{
		Current:  current,
		Previous: previous,
	}
	add := func(da DeltaAction) {
		db.Actions = append(db.Actions, da)
	}
	// Calculate node and field level actions.
	// List through all nodes and first determine node removals, additions, and renames.
	// Then go through remaining node that remain from version to version
	// to determine field addtions , removals, then renames.
	// Lastly determine field level updates.
	//
	// Pre-additions scripts.
	// Node additions.
	// Node renames
	// Field additions.
	// Field Renames.
	// Field Updates.
	// Field Removals.
	// Node Removals.
	// Post-removals scripts.

	type NodeCP struct {
		Current  *Node
		Previous *Node
	}
	// Node additions and node renames.
	nodeCurrentWithPrevious := make([]NodeCP, 0, len(db.Current.Nodes))
	for ni := range db.Current.Nodes {
		nc := &db.Current.Nodes[ni]
		np := db.Previous.Node(nc.Name)
		if np == nil {
			add(DeltaAction{
				Alter:       AlterNodeAdd,
				NodeCurrent: nc,
			})
			continue
		}
		nodeCurrentWithPrevious = append(nodeCurrentWithPrevious, NodeCP{
			Current:  nc,
			Previous: np,
		})
		if nc.Name != np.Name {
			add(DeltaAction{
				Alter:        AlterNodeRename,
				NodeCurrent:  nc,
				NodePrevious: np,
			})
		}
	}

	// Field additions and
	for _, cp := range nodeCurrentWithPrevious {
		for ri := range cp.Current.Roles {
			rc := &cp.Current.Roles[ri]
			rp := cp.Previous.Role(rc.Name)
			for fi := range rc.Fields {
				fc := &rc.Fields[fi]
				var fp *Field
				if rp == nil {
					add(DeltaAction{
						Alter:        AlterFieldAdd,
						NodeCurrent:  cp.Current,
						NodePrevious: cp.Previous,
						FieldCurrent: fc,
					})
					continue
				}
				if fc.ID > 0 {
					fp = rp.fieldIDLookup[fc.ID]
				}
				if fp == nil {
					fp = rp.fieldNameLookup[fc.name]
				}
				if fp == nil {
					add(DeltaAction{
						Alter:        AlterFieldAdd,
						NodeCurrent:  cp.Current,
						NodePrevious: cp.Previous,
						FieldCurrent: fc,
					})
					continue
				}
				if fc.name != fp.name {
					add(DeltaAction{
						Alter:         AlterFieldRename,
						NodeCurrent:   cp.Current,
						NodePrevious:  cp.Previous,
						FieldCurrent:  fc,
						FieldPrevious: fp,
					})
				}
				if fc.needUpdate(fp) {
					add(DeltaAction{
						Alter:         AlterFieldUpdate,
						NodeCurrent:   cp.Current,
						NodePrevious:  cp.Previous,
						FieldCurrent:  fc,
						FieldPrevious: fp,
					})
				}
			}
		}
		// Field removals.
		for ri := range cp.Previous.Roles {
			rp := &cp.Previous.Roles[ri]
			rc := cp.Current.Role(rp.Name)
			for fi := range rp.Fields {
				fp := &rp.Fields[fi]

				if rc == nil {
					add(DeltaAction{
						Alter:         AlterFieldRemove,
						NodeCurrent:   cp.Current,
						NodePrevious:  cp.Previous,
						FieldPrevious: fp,
					})
					continue
				}

				var fc *Field
				if fp.ID > 0 {
					fc = rc.fieldIDLookup[fp.ID]
				}
				if fc == nil {
					fc = rc.fieldNameLookup[fp.name]
				}
				if fc == nil {
					add(DeltaAction{
						Alter:         AlterFieldRemove,
						NodeCurrent:   cp.Current,
						NodePrevious:  cp.Previous,
						FieldPrevious: fp,
					})
					continue
				}
			}
		}
	}
	if db.Previous == nil {
		return db, nil
	}
	// Node removals.
	for ni := range db.Previous.Nodes {
		np := &db.Previous.Nodes[ni]
		nc := db.Current.Node(np.Name)
		if nc == nil {
			add(DeltaAction{
				Alter:        AlterNodeRemove,
				NodePrevious: np,
			})
			continue
		}
	}
	return db, nil
}

func (*DeltaBus) String() string {
	panic("TODO")
}
