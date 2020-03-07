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
	groupList
)

type parseLine struct {
	Parent     *parseLine
	Group      groupType
	Identifier parseIdentifier
	Index      int64
	Value      parseIdentifier
}

func (line *parseLine) String() string {
	sb := &strings.Builder{}
	chain := make([]*parseLine, 0, 4)
	cp := line
	for cp != nil {
		chain = append(chain, cp)
		cp = cp.Parent
	}
	for i := len(chain) - 1; i >= 0; i-- {
		if sb.Len() > 0 {
			sb.WriteRune('.')
		}
		cp := chain[i]
		cp.Identifier.write(sb)
	}
	sb.WriteRune('[')
	sb.WriteString(strconv.FormatInt(line.Index, 10))
	sb.WriteRune(']')

	sb.WriteString(" = ")
	line.Value.write(sb)

	return sb.String()
}

type parseIdentifier struct {
	Parts []lexToken
	Dot   bool
}

func (id parseIdentifier) canAppend() bool {
	return len(id.Parts) == 0 || id.Dot
}
func (id parseIdentifier) empty() bool {
	return len(id.Parts) == 0 && id.Dot == false
}
func (id parseIdentifier) write(sb *strings.Builder) {
	for i, p := range id.Parts {
		if i != 0 {
			sb.WriteRune('.')
		}
		sb.WriteString(p.Value)
	}
}

type lineEmitter struct {
	Root    *parseLine
	Current *parseLine
}

func (e *lineEmitter) EmitToken(lt lexToken) error {
	fmt.Printf("\tz: %v\n", lt)

	if e.Current == nil {
		e.Current = &parseLine{
			Parent: e.Root,
			Group:  e.Root.Group,
		}
	}

	switch e.Current.Group {
	default:
		return fmt.Errorf("unknown group type: %v", e.Current.Group)
	case groupStruct:
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
		*/
		switch lt.Type {
		default:
			return terr("unknown struct token type", lt)
		case tokenEOF:
			return io.EOF
		case tokenNewline, tokenEOS:
			return nil
		case tokenWhitespace:
			return nil
		case tokenIdentifier, tokenQuote, tokenNumber:
			switch {
			case e.Current.Identifier.canAppend():
				e.Current.Identifier.Parts = append(e.Current.Identifier.Parts, lt)
				e.Current.Identifier.Dot = false
				return nil
			case e.Current.Value.canAppend():
				e.Current.Value.Parts = append(e.Current.Value.Parts, lt)
				e.Current.Value.Dot = false
				return nil
			default:
				next := &parseLine{
					Parent:     e.Current.Parent,
					Index:      e.Current.Index + 1,
					Group:      e.Current.Group,
					Identifier: e.Current.Identifier,
					Value:      parseIdentifier{Parts: []lexToken{lt}},
				}
				e.EmitLine(e.Current)
				e.Current = next
				return nil
			}
		case tokenComment:
			return nil
		case tokenSymbol:
			switch lt.Value {
			default:
				return terr("unknown struct token symbol", lt)
			case ",":
				next := &parseLine{
					Parent:     e.Current.Parent,
					Group:      e.Current.Group,
					Identifier: e.Current.Identifier,
				}
				e.EmitLine(e.Current)
				e.Current = next
				return nil
			case ".":
				switch {
				default:
					return terr("unexpected '.'", lt)
				case e.Current.Value.empty() && !e.Current.Identifier.empty():
					e.Current.Identifier.Dot = true
				case !e.Current.Value.empty():
					e.Current.Value.Dot = true
				}
				return nil
			case "{":
				next := &parseLine{
					Parent: e.Current,
					Group:  groupStruct,
				}
				e.EmitLine(e.Current)
				e.Current = next
				return nil
			case "}":
				parent := e.Current.Parent
				if parent.Parent == nil {
					return terr("extra '}'", lt)
				}
				e.EmitLine(e.Current)
				e.Current = parent
				return nil
			case "(":
				return nil
			case ")":
				return nil
			}
		}
	case groupList:
		panic("TODO: List")
	}
}

func (e *lineEmitter) EmitLine(line *parseLine) error {
	fmt.Printf("a: %v\n", line)
	return nil
}
