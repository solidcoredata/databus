package parse

import (
	"fmt"
	"strings"
)

type Valuer interface {
	Value() Valuer
	Children() []Valuer
	Index() int32
	Span() (start Pos, end Pos)
}

type VI struct {
	Valuer
	val        Valuer
	index      int32
	start, end Pos
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
func (vi *VI) Span() (Pos, Pos) {
	if vi == nil {
		return Pos{}, Pos{}
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

	start, end Pos
}

func (ql *QueryLine) Value() Valuer {
	return ql
}
func (ql *QueryLine) Index() int32 {
	return 0
}
func (ql *QueryLine) Span() (Pos, Pos) {
	if ql == nil {
		return Pos{}, Pos{}
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
	start, end Pos
}

func (q *Query) Value() Valuer {
	return q
}
func (q *Query) Index() int32 {
	return 0
}
func (q *Query) Span() (Pos, Pos) {
	if q == nil {
		return Pos{}, Pos{}
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
	type ParseContext struct {
		Context     *parseFullIdentifier
		ContextText string
		File        *file
	}
	var pctx ParseContext

	for _, st := range pr.Statements {
		if pctx.File != st.Start.File {
			pctx = ParseContext{
				File: st.Start.File,
			}
		}
		if st.Type == statementContext {
			pctx.Context = st.Identifier
			buf := &strings.Builder{}
			buf.WriteString("(")
			pctx.Context.WriteToBuilder(buf, 0)
			buf.WriteString(") ")
			pctx.ContextText = buf.String()
			continue
		}
		if st.Value == nil {
			fmt.Printf("%v %s%v\n", st.Type, pctx.ContextText, st.Identifier)
			continue
		}
		for vi, v := range st.Value.Values {
			if v.Token.Type != tokenUnknown {
				fmt.Printf("%v %s%v [%d]%v (%v)\n", st.Type, pctx.ContextText, st.Identifier, vi, v.Token.Value, v.Token.Type)
			}
			if v.Table == nil {
				continue
			}
			t := v.Table
			vv := &strings.Builder{}
			cnames := []string{}
			for ri, row := range t.Rows {
				key := ""
				for ci, cell := range row.Cells {
					_, _ = ci, ri
					vv.Reset()
					cell.WriteToBuilder(vv, 0)
					valueText := vv.String()
					if ri == 0 {
						cnames = append(cnames, valueText)
						continue
					}
					if ci == 0 {
						key = valueText
					}
					if len(cnames) >= ci {
						// TODO(daniel.theophanes): Add a parse error.
						continue
					}
					cn := cnames[ci]
					fmt.Printf("%v %s%v.%s.%s [%d]%v (%v) %v\n", st.Type, pctx.ContextText, st.Identifier, key, cn, vi, v.Token.Value, v.Token.Type, vv.String())
				}
			}
		}
	}
	return nil, nil
}
