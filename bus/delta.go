package bus

func NewDelta(current, previous *Bus) (*DeltaBus, error) {
	db := &DeltaBus{
		Current:  current.Version,
		Previous: previous.Version,
	}
	return db, nil
}

func (b *Bus) NodesTopological() []*Node {
	return nil
}

func (*DeltaBus) String() string {
	panic("TODO")
}
