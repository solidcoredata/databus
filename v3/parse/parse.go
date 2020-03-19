package parse

import (
	"io"
	"strconv"
	"strings"
)

//go:generate stringer -trimprefix group -type groupType

type groupType int

const (
	groupUnknown   groupType = iota
	groupStruct              // At the "{".
	groupStructKey           // First identifier in a statement within a struct.
	groupList                // At the "(".
	groupValue               // Any valued object.
	groupTable               // At the "{|"
	groupTableHead           // First line of the table.
	groupTableData           // All lines except the first line.
)

type parseLine struct {
	Parent     *parseLine
	LastChild  *parseLine
	Index      int64
	Group      groupType
	Identifier parseIdentifier
	Sent       bool

	Header []parseIdentifier
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
	if indent && len(chain) > 1 {
		sb.WriteString(strings.Repeat("\t", len(chain)-1))
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

// EmitToken takes a sequence of tokens and turns them into parseLines.
func (e *lineEmitter) EmitToken(lt lexToken) error {
	// fmt.Printf("z: %v\n", lt)

	if e.Current == nil {
		e.Current = e.Root
	}

	switch lt.Type {
	default:
		return terr("unknown struct token type", lt)
	case tokenEOF:
		if err := e.EmitLine(e.Current); err != nil {
			return err
		}
		return io.EOF
	case tokenEOS:
		if err := e.EmitLine(e.Current); err != nil {
			return err
		}
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
			if err := e.EmitLine(next); err != nil {
				return err
			}

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
			if err := e.EmitLine(next); err != nil {
				return err
			}

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
			if err := e.EmitLine(next); err != nil {
				return err
			}

			e.Current = next
			return nil
		}
	case tokenComment:
		return nil
	case tokenSymbol:
		switch lt.Value {
		default:
			return terr("unknown symbol type", lt)
		case ",":
			if err := e.EmitLine(e.Current); err != nil {
				return err
			}
			return nil
		case ".":
			if !e.Current.Identifier.canAddDot() {
				return terr(`unexpected "."`, lt)
			}
			e.Current.Identifier.Dot = true
			return nil
		case "{":
			// if err := e.EmitLine(e.Current); err != nil {
			// 	return err
			// }
			next := &parseLine{
				Parent: e.Current.Parent,
				Group:  groupStruct,
				Index:  e.Current.Index + 1,
			}
			if err := e.EmitLine(next); err != nil {
				return err
			}
			e.Current = next
			return nil
		case "(":
			// if err := e.EmitLine(e.Current); err != nil {
			// 	return err
			// }
			next := &parseLine{
				Parent: e.Current.Parent,
				Group:  groupList,
				Index:  e.Current.Index + 1,
			}
			if err := e.EmitLine(next); err != nil {
				return err
			}
			e.Current = next
			return nil
		case "}":
			look := e.Current
			for look != nil {
				switch look.Group {
				default:
					panic("unknown group type")
				case groupStruct:
					e.Current = look.Parent
					if e.Current != nil && e.Current.Parent != nil {
						e.Current = e.Current.Parent
					}
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
		case "|":
			switch e.Current.Group {
			default:
				panic("unknown group type")
			case groupStruct: // Transition to table header.
				e.Current.Group = groupTable
			case groupTable:
				return terr(`unexpected "|"`, lt)
			case groupValue:
				parent := e.Current.Parent
				switch parent.Group {
				default:
					panic("unknown group type")
				case groupTable:
					return terr("unexpected value after table expected EOS", lt)
				case groupTableHead:
					parent.Header = append(parent.Header, e.Current.Identifier)
				case groupTableData:
					//
				}
			}
			return nil
		}
	}
}

func (e *lineEmitter) EmitLine(line *parseLine) error {
	// Never emit the file struct.
	if line.Parent == nil {
		return nil
	}
	if line.Sent == true {
		// return fmt.Errorf("line sent twice: %v", line)
		return nil
	}
	line.Sent = true

	e.all = append(e.all, line)
	// fmt.Println(line)
	return nil
}
