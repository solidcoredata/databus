package parse

import "fmt"

type Valuer interface {
	Value() Valuer
	Children() []Valuer
	Index() int32
	Span() (start pos, end pos)
}

type VI struct {
	Valuer
	val        Valuer
	index      int32
	start, end pos
}

func (vi *VI) Value() Valuer {
	if vi == nil {
		return nil
	}
	return vi.val
}
func (vi *VI) Index() int32 {
	if vi == nil {
		return 0
	}
	return vi.index
}
func (vi *VI) Span() (pos, pos) {
	if vi == nil {
		return pos{}, pos{}
	}
	return vi.start, vi.end
}
func (vi *VI) Children() []Valuer {
	return nil
}

type QueryLine struct {
	Valuer
	Verb   string // from, select, and.
	Values []Valuer

	start, end pos
}

func (ql *QueryLine) Value() Valuer {
	return ql
}
func (ql *QueryLine) Index() int32 {
	return 0
}
func (ql *QueryLine) Span() (pos, pos) {
	if ql == nil {
		return pos{}, pos{}
	}
	return ql.start, ql.end
}
func (ql *QueryLine) Children() []Valuer {
	if ql == nil {
		return nil
	}
	return ql.Values
}

// Query must itself implement the valuer interface.
type Query struct {
	Valuer
	Lines      []*QueryLine
	start, end pos
}

func (q *Query) Value() Valuer {
	return q
}
func (q *Query) Index() int32 {
	return 0
}
func (q *Query) Span() (pos, pos) {
	if q == nil {
		return pos{}, pos{}
	}
	return q.start, q.end
}
func (q *Query) Children() []Valuer {
	if q == nil {
		return nil
	}
	vv := make([]Valuer, len(q.Lines))
	for i, l := range q.Lines {
		vv[i] = l
	}
	return vv
}

type Root struct {
	FullPath map[string]Valuer
}

func Parse4(pr *parseRoot) (*Root, error) {
	for _, st := range pr.Statements {
		fmt.Printf("%v %v %v\n", st.Type, st.Identifier, st.Value)

	}
	return nil, nil
}
