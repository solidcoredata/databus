package parse

import (
	"io"
	"strconv"
	"strings"
)

//go:generate stringer -trimprefix group -type groupType

type groupType int

const (
	groupUnknown     groupType = iota
	groupStruct                // At the "{".
	groupStructKey             // First identifier in a statement within a struct.
	groupStructValue           // Struct Value, after the key.
	groupList                  // At the "(".
	groupListValue             // Any valued object.
	groupTable                 // At the "{|"
	groupTableHead             // First line of the table.
	groupTableData             // All lines except the first line.
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

type match struct {
	token lexToken
}

func (m *match) Value() bool {
	// Value = tokenIdentifier, tokenNumber, tokenText, ".", "(", "{".
	switch m.token.Type {
	default:
		return false
	case tokenIdentifier, tokenNumber, tokenQuote:
		return true
	case tokenSymbol:
		switch m.token.lexTokenValue {
		default:
			return false
		case vPeriod:
			return true
		case vListStart:
			return true
		case vStructStart:
			return true
		}
	}
}

func (m *match) Identifier() bool {
	// Identifier = tokenIdentifier, tokenNumber, tokenText
	switch m.token.Type {
	default:
		return false
	case tokenIdentifier, tokenNumber, tokenQuote:
		return true
	}
}

func (m *match) Comma() bool {
	switch m.token.lexTokenValue {
	default:
		return false
	case vComma:
		return true
	}
}

func (m *match) ListEnd() bool {
	switch m.token.lexTokenValue {
	default:
		return false
	case vListEnd:
		return true
	}
}
func (m *match) StructEnd() bool {
	switch m.token.lexTokenValue {
	default:
		return false
	case vStructEnd:
		return true
	}
}
func (m *match) EOS() bool {
	switch m.token.lexTokenValue {
	default:
		return false
	case vEOS:
		return true
	}
}

type lineEmitter struct {
	Root    *parseLine
	Current *parseLine

	all []*parseLine
}

func (e *lineEmitter) eatIdentifier(lt lexToken, newValueGroup groupType) (next *parseLine, err error) {
	switch lt.Type {
	default:
		panic("invalid state: mismatch between consume and match Type")
	case tokenSymbol:
		switch lt.lexTokenValue {
		default:
			panic("invalid state: mismatch between consume and match TokenValue")
		case vPeriod:
			switch {
			default:
				err = terr(`unexpected "."`, lt)
				return
			case e.Current.Identifier.canAddDot():
				e.Current.Identifier.Dot = true
			}
		case vStructStart:
			next = e.Current
			e.Current = &parseLine{
				Parent: next.Parent,
				Index:  next.Index + 1,
				Group:  groupStruct,
			}
		case vListStart:
			next = e.Current
			e.Current = &parseLine{
				Parent: next.Parent,
				Index:  next.Index + 1,
				Group:  groupList,
			}
		}
	case tokenIdentifier, tokenNumber, tokenQuote:
		switch {
		default:
			// New value under same parent.
			next = e.Current
			e.Current = &parseLine{
				Parent:     next.Parent,
				Index:      next.Index + 1,
				Group:      newValueGroup, // Unique in setup.
				Identifier: parseIdentifier{Parts: []lexToken{lt}},
			}
		case e.Current.Identifier.canAppend():
			e.Current.Identifier.Parts = append(e.Current.Identifier.Parts, lt)
			e.Current.Identifier.Dot = false
		}
	}
	return
}

// EmitToken takes a sequence of tokens and turns them into parseLines.
func (e *lineEmitter) EmitToken(lt lexToken) error {
	// Identifier = tokenIdentifier, tokenNumber, tokenText
	// Value = tokenIdentifier, tokenNumber, tokenText, ".", "(", "{".

	if lt.lexTokenValue == vEOF {
		if e.Current == e.Root {
			return io.EOF
		}
		return terr("unexpected end of file", lt)
	}

	m := &match{
		token: lt,
	}
	var (
		next *parseLine
		err  error
	)

	switch e.Current.Group {
	default:
		panic("uknown group type")
	case groupList:
		// Value, ")", EOS.
		switch {
		default:
			return terr("unexpected list token", lt)
		case m.Value():
			next = e.Current
			e.Current = &parseLine{
				Parent:     next,
				Group:      groupListValue,
				Identifier: parseIdentifier{Parts: []lexToken{lt}},
			}
		case m.ListEnd():
		case m.EOS():
		}
	case groupListValue:
		// Value, ",", ")", EOS.
		switch {
		default:
			return terr("unexpected list token", lt)
		case m.Value():
			next, err = e.eatIdentifier(lt, groupListValue)
		case m.Comma(), m.EOS():
		case m.ListEnd():
		}
	case groupStruct:
		// Identifier, "}", "|".
	case groupStructKey:
		// Value.
	case groupStructValue:
		// Value, ",", "}" EOS.
	case groupTable:
		// Identifier, "}", EOS.
	case groupTableHead:
		// Identifier, "}", "|", EOS.
	case groupTableData:
		// Value, "}", "|", EOS.
	}

	e.EmitLine(next)

	return err
}

func (e *lineEmitter) EmitLine(line *parseLine) error {
	if line == nil {
		return nil
	}
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
