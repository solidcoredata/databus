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
func (m *match) Pipe() bool {
	switch m.token.lexTokenValue {
	default:
		return false
	case vPipe:
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
	Build   *parseLine

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
			case e.Build.Identifier.canAddDot():
				e.Build.Identifier.Dot = true
			}
		case vStructStart:
			e.EmitLine(e.Build)
			parent := e.Current.Parent
			e.Current = &parseLine{
				Parent: parent,
				Group:  groupStruct,
			}
			if parent.LastChild != nil {
				e.Current.Index = parent.LastChild.Index + 1
			}
			next = e.Current
		case vListStart:
			e.EmitLine(e.Build)
			parent := e.Current.Parent
			e.Current = &parseLine{
				Parent: parent,
				Group:  groupList,
			}
			if parent.LastChild != nil {
				e.Current.Index = parent.LastChild.Index + 1
			}
			next = e.Current
		}
	case tokenIdentifier, tokenNumber, tokenQuote:
		switch {
		default:
			// New value under same parent.
			if e.Current == nil {
				panic("nil current")
			}
			if e.Current.Parent == nil {
				panic("nil current parent")
			}
			ListOrStructKey := e.Current.Parent
			switch ListOrStructKey.Group {
			default:
				panic("unexpected group type")
			case groupList:
				err = terr("unexpected token in list", lt)
			case groupStructKey:
				next = e.Build
				e.Build = &parseLine{
					Parent:     ListOrStructKey,
					Group:      groupStructValue,
					Identifier: parseIdentifier{Parts: []lexToken{lt}},
				}
				if ListOrStructKey.LastChild != nil {
					e.Build.Index = ListOrStructKey.LastChild.Index + 1
				}
				ListOrStructKey.LastChild = e.Build
			}

		case e.Build.Identifier.canAppend():
			e.Build.Identifier.Parts = append(e.Build.Identifier.Parts, lt)
			e.Build.Identifier.Dot = false
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
		panic("uknown group type " + e.Current.Group.String())
	case groupList:
		// Value, ")", EOS.
		switch {
		default:
			return terr("unexpected list token", lt)
		case m.Value():
			list := e.Current
			e.Build = &parseLine{
				Parent:     list,
				Group:      groupListValue,
				Identifier: parseIdentifier{Parts: []lexToken{lt}},
			}
			e.Current = e.Build
			if list.LastChild != nil {
				e.Build.Index = list.LastChild.Index + 1
			}
			list.LastChild = e.Current
		case m.ListEnd():
			e.Current = e.Current.Parent
		}
	case groupListValue:
		// Value, ",", ")", EOS.
		switch {
		default:
			return terr("unexpected list token", lt)
		case m.Value():
			next, err = e.eatIdentifier(lt, groupListValue)
		case m.Comma(), m.EOS():
			if e.Build == nil {
				return terr(`unexpected "," in list`, lt)
			}
			next = e.Build
			e.Build = nil
			e.Current = e.Current.Parent
		case m.ListEnd():
			next = e.Build
			e.Build = nil
			value := e.Current
			list := value.Parent
			e.Current = list.Parent.Parent
		}
	case groupStruct:
		// Identifier, "}", "|".
		switch {
		default:
			return terr("unexpected struct token", lt)
		case m.Identifier():
			gStruct := e.Current
			e.Current = &parseLine{
				Parent:     gStruct,
				Group:      groupStructKey,
				Identifier: parseIdentifier{Parts: []lexToken{lt}},
			}
			if gStruct.LastChild != nil {
				e.Current.Index = gStruct.LastChild.Index + 1
			}
			gStruct.LastChild = e.Current
			next = e.Current
		case m.StructEnd():
			e.Current = e.Current.Parent.Parent
		case m.Pipe():
			e.Current.Group = groupTable
		}
	case groupStructKey:
		// Value, "}".
		switch {
		default:
			return terr("unexpected struct key token", lt)
		case m.Value():
			key := e.Current
			e.Current = &parseLine{
				Parent:     e.Current,
				Group:      groupStructValue,
				Identifier: parseIdentifier{Parts: []lexToken{lt}},
			}
			key.LastChild = e.Current
			e.Build = e.Current
		case m.StructEnd():
			gStruct := e.Current.Parent
			e.Current = gStruct.Parent.Parent
		}
	case groupStructValue:
		// Value, ",", "}" EOS.
		switch {
		default:
			return terr("unexpected struct value token", lt)
		case m.Value():
			next, err = e.eatIdentifier(lt, groupStructValue)
		case m.Comma():
			e.EmitLine(e.Build)
			key := e.Current.Parent
			e.Current = &parseLine{
				Parent:     key.Parent,
				Group:      key.Group,
				Index:      key.Index + 1,
				Identifier: key.Identifier,
			}
			key.Parent.LastChild = e.Current
			next = e.Current
		case m.EOS():
			next = e.Build
			e.Build = nil

			key := e.Current.Parent
			gStruct := key.Parent
			e.Current = gStruct
		case m.StructEnd():
			next = e.Build
			e.Build = nil

			key := e.Current.Parent
			gStruct := key.Parent
			e.Current = gStruct.Parent
		}
	case groupTable:
		// Identifier, "}", EOS.
		switch {
		default:
			return terr("unexpected tabke token", lt)
		case m.Identifier():
			e.Current.Header = append(e.Current.Header, parseIdentifier{Parts: []lexToken{lt}})
			e.Current.Group = groupTableHead
		case m.StructEnd():
			// TODO
		case m.EOS():
			e.Current.Group = groupTableData
		}
	case groupTableHead:
		// Identifier, "}", "|", EOS.
	case groupTableData:
		// Value, "}", "|", EOS.
	}
	if err != nil {
		return err
	}

	return e.EmitLine(next)
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
	fmt.Println(line)
	return nil
}
