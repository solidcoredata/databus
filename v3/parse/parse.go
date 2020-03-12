package parse

import (
	"fmt"
	"io"
	"strconv"
	"strings"
)

//go:generate stringer -trimprefix group -type groupType

type groupType int

const (
	groupUnknown groupType = iota
	groupStruct
	groupStructKey
	groupList
	groupValue
)

type parseLine struct {
	Parent     *parseLine
	LastChild  *parseLine
	Index      int64
	Group      groupType
	Identifier parseIdentifier
}

func (line *parseLine) String() string {
	s := line.baseString(true)
	if false && line.LastChild != nil {
		s += " LC=" + line.LastChild.baseString(false)
	}

	return s
}

func (line *parseLine) baseString(indent bool) string {
	sb := &strings.Builder{}
	chain := make([]*parseLine, 0, 4)
	cp := line.Parent
	for cp != nil {
		chain = append(chain, cp)
		cp = cp.Parent
	}
	if indent {
		sb.WriteString(strings.Repeat("\t", len(chain)-2))
	}

	if false {
		var n int
		for i := len(chain) - 1; i >= 0; i-- {
			if n > 0 {
				sb.WriteRune('.')
			}

			cp := chain[i]
			n = cp.Identifier.write(sb)
		}
		sb.WriteRune('[')
		sb.WriteString(strconv.FormatInt(line.Index, 10))
		sb.WriteRune(']')

		if !line.Identifier.empty() {
			sb.WriteString(" = ")
			line.Identifier.write(sb)
		}
	} else {
		sb.WriteRune('[')
		sb.WriteString(strconv.FormatInt(line.Index, 10))
		sb.WriteRune(']')
		line.Identifier.write(sb)
	}

	switch line.Group {
	case groupStruct:
		sb.WriteString("(struct)")
	case groupStructKey:
		sb.WriteString("(key)")
	case groupList:
		sb.WriteString("(list)")
	}
	return sb.String()
}

type parseIdentifier struct {
	Parts []lexToken
	Dot   bool
}

// canAppend returns true if it is valid to append another lexToken to the existing identifier.
func (id parseIdentifier) canAppend() bool {
	return len(id.Parts) == 0 || (id.Dot && len(id.Parts) > 0)
}

// canAddDot returns true if it is valid to add a "dot" to the identifier.
func (id parseIdentifier) canAddDot() bool {
	return !id.Dot && len(id.Parts) > 0
}
func (id parseIdentifier) empty() bool {
	return len(id.Parts) == 0 && id.Dot == false
}
func (id parseIdentifier) write(sb *strings.Builder) int {
	var n, total int
	for i, p := range id.Parts {
		if i != 0 {
			n, _ = sb.WriteRune('.')
			total += n
		}
		n, _ = sb.WriteString(p.Value)
		total += n
	}
	return total
}

type lineEmitter struct {
	Root    *parseLine
	Current *parseLine

	all []*parseLine
}

/*
	EmitToken is called each time a new lexToken is parsed.
	When sufficent tokens have been parsed a new ParseLine sent to EmitLine.


	set SimpleSelect query{
		from books b
		from genre g
		and eq(b.published, true)
		select b.id, b.bookname
		select g.name genrename
		and eq(b.deleted, false)
	}

	Parent file
	Index 0
	Group struct
	Identifier set

	Parent set
	Index 0
	Identifier SimpleSelect

	Parent set
	Index 1
	Identifier query

	Parent query
	Index 0
	Identifier from

	Parent from
	Index 0
	Identifier books

	Parent from
	Index 1
	Identifier b

*/
func (e *lineEmitter) EmitToken(lt lexToken) error {
	fmt.Printf("z: %v\n", lt)

	if e.Current == nil {
		e.Current = &parseLine{
			Parent: e.Root,
			Group:  e.Root.Group,
		}
	}

	/*
		tokenNewline
		tokenWhitespace
		tokenIdentifier
		tokenComment
		tokenSymbol
		tokenNumber
		tokenQuote
		tokenEOF
		tokenEOS

		switch e.Current.Group {
		default:
			return fmt.Errorf("unknown group type: %v", e.Current.Group)
		case groupStruct:
			return nil
		case groupList:
			return nil
		case groupValue:
			return nil
		}
	*/
	switch lt.Type {
	default:
		return terr("unknown struct token type", lt)
	case tokenEOF:
		e.printAll()
		return io.EOF
	case tokenEOS:
		switch e.Current.Group {
		default:
			panic("unknown group type")
		case groupStruct, groupList, groupStructKey:
			return nil
		case groupValue:
			e.Current = e.Current.Parent.Parent
			return nil
		}
	case tokenIdentifier, tokenQuote, tokenNumber:
		switch e.Current.Group {
		default:
			panic("unknown group type")
		case groupStruct:
			next := &parseLine{
				Parent:     e.Current,
				Group:      groupStructKey,
				Identifier: parseIdentifier{Parts: []lexToken{lt}},
			}
			if e.Current.LastChild != nil {
				next.Index = e.Current.LastChild.Index + 1
			}
			e.Current.LastChild = next
			e.EmitLine(next)

			e.Current = next
			return nil
		case groupList:
			next := &parseLine{
				Parent:     e.Current,
				Group:      groupValue,
				Identifier: parseIdentifier{Parts: []lexToken{lt}},
			}
			if e.Current.LastChild != nil {
				next.Index = e.Current.LastChild.Index + 1
			}
			e.Current.LastChild = next
			e.EmitLine(next)

			e.Current = next
			return nil
		case groupStructKey:
			lastChild := e.Current.LastChild
			next := &parseLine{
				Parent:     e.Current,
				Group:      groupValue,
				Identifier: parseIdentifier{Parts: []lexToken{lt}},
			}
			if lastChild != nil {
				next.Index = lastChild.Index + 1
			}
			e.Current.LastChild = next
			e.EmitLine(next)

			e.Current = next
			return nil
		case groupValue:
			if e.Current.Identifier.canAppend() {
				e.Current.Identifier.Parts = append(e.Current.Identifier.Parts, lt)
				e.Current.Identifier.Dot = false
				return nil
			}
			parentLastChild := e.Current.Parent.LastChild
			next := &parseLine{
				Parent:     e.Current.Parent,
				Group:      groupValue,
				Identifier: parseIdentifier{Parts: []lexToken{lt}},
			}
			if parentLastChild != nil {
				next.Index = parentLastChild.Index + 1
			}
			e.Current.Parent.LastChild = next
			e.EmitLine(next)

			e.Current = next
			return nil
		}
	case tokenComment:
		return nil
	case tokenSymbol:
		switch lt.Value {
		default:
			return nil
		case ".":
			if !e.Current.Identifier.canAddDot() {
				return terr(`unexpected "."`, lt)
			}
			e.Current.Identifier.Dot = true
			return nil
		case "{":
			next := &parseLine{
				Parent: e.Current.Parent,
				Group:  groupStruct,
				Index:  e.Current.Index + 1,
			}
			e.EmitLine(next)
			e.Current = next
			return nil
		case "(":
			next := &parseLine{
				Parent: e.Current.Parent,
				Group:  groupList,
				Index:  e.Current.Index + 1,
			}
			e.EmitLine(next)
			e.Current = next
			return nil
		case "}":
			look := e.Current
			for look != nil {
				switch look.Group {
				default:
					panic("unknown group type")
				case groupStruct:
					e.Current = look.Parent.Parent
					return nil
				case groupList:
					return terr("expected '}' not ')'", lt)
				case groupStructKey, groupValue:
					// Nothing.
				}
				look = look.Parent
			}
			return nil
		case ")":
			look := e.Current
			for look != nil {
				switch look.Group {
				default:
					panic("unknown group type")
				case groupStruct:
					return terr("expected ')' not '}'", lt)
				case groupList:
					e.Current = look.Parent.Parent
					return nil
				case groupStructKey, groupValue:
					// Nothing.
				}
				look = look.Parent
			}
			return nil
		}
	}
}

func (e *lineEmitter) EmitLine(line *parseLine) error {
	e.all = append(e.all, line)
	fmt.Println(line)
	return nil
}

func (e *lineEmitter) printAll() {
	for _, line := range e.all {
		fmt.Println(line)
	}
}
