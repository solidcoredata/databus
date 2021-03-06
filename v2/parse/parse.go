package parse

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"strings"
	"unicode"
	"unicode/utf8"
)

/*
// This is a testing ground for table definitions.

verbs:
create <identifier>
context <identifier>
set <identifier> <value>

value: array(a, b, c, ...)
value: table(
    headerA | headerB
    value1A | vlaue1B
    value2A | value2B
)

// Comment to end of line. No multi-line comments yet.
*/

type FileReader interface {
	Load(loader FileLoader) error
}

type FileLoader func(name string, r io.Reader) error

type file struct {
	Name string
}
type Pos struct {
	File     *file
	Line     int64 // line in input (starting at 1)
	LineRune int64 // rune in line (starting at 1)
	Byte     int64 // byte in input (starting at 0)
}

type state struct {
	ctx  context.Context
	stop func()
	err  error

	files []*file

	buf *bufio.Reader
	pos Pos

	nextBuf []rune
	out     *strings.Builder

	emitter  chan lexToken
	comments *parseCommentBlock
	prevEmit lexToken
}

func (s *state) load(name string, r io.Reader) error {
	f := &file{Name: name}
	s.files = append(s.files, f)
	s.buf = bufio.NewReader(r)
	s.out = &strings.Builder{}
	s.pos = Pos{
		File:     f,
		Line:     1,
		LineRune: 1,
		Byte:     0,
	}

	err := s.runLexer()
	if err == io.EOF {
		s.emitToken(lexToken{Start: s.pos, Type: tokenEOF})
		return nil
	}
	if err != nil {
		return fmt.Errorf("%s:%d:%d %v", s.pos.File.Name, s.pos.Line, s.pos.LineRune, err)
	}
	return err
}

func ParseFile(ctx context.Context, fr FileReader) (*parseRoot, error) {
	ctx, cancel := context.WithCancel(ctx)
	s := &state{
		ctx:     ctx,
		stop:    cancel,
		emitter: make(chan lexToken, 3),
	}
	defer cancel()

	root := &parseRoot{}
	done := make(chan error, 1)
	// Concurrent only to make it easier to program for the time being.
	go func() {
		done <- s.runParse(root)
		cancel()
	}()

	err := fr.Load(s.load)
	close(s.emitter)
	if err != nil {
		if cerr, _ := <-done; cerr != nil {
			return nil, cerr
		}
		return nil, err
	}

	return root, <-done
}

type lexRune func(r rune) bool

type lex struct {
	is         tokenType
	test       lexRune
	next       []lex
	while      lexRune
	end        []lexRune
	excludeEnd bool
}

//go:generate stringer -trimprefix token -type tokenType

type tokenType int

const (
	tokenUnknown tokenType = iota
	tokenNewline
	tokenWhitespace
	tokenIdentifier
	tokenComment
	tokenSymbol
	tokenNumber
	tokenQuote
	tokenEOF
	tokenEOS
)

var lexRoot = []lex{
	{
		is: tokenNewline,
		test: func(r rune) bool {
			return r == '\n'
		},
		while: func(r rune) bool {
			return false
		},
	},
	{
		is: tokenWhitespace,
		test: func(r rune) bool {
			return unicode.IsSpace(r)
		},
		while: func(r rune) bool {
			return unicode.IsSpace(r)
		},
	},
	{
		is: tokenIdentifier,
		test: func(r rune) bool {
			return unicode.IsLetter(r) || r == '_'
		},
		while: func(r rune) bool {
			return unicode.IsLetter(r) || unicode.IsDigit(r) || r == '_'
		},
	},
	{
		test: func(r rune) bool {
			return r == '/'
		},
		next: []lex{
			{
				is: tokenComment,
				test: func(r rune) bool {
					return r == '/'
				},
				end: []lexRune{
					func(r rune) bool {
						return r == '\n'
					},
				},
				excludeEnd: true,
			},
			{
				is: tokenComment,
				test: func(r rune) bool {
					return r == '*'
				},
				end: []lexRune{
					func(r rune) bool {
						return r == '*'
					},
					func(r rune) bool {
						return r == '/'
					},
				},
			},
		},
		while: func(r rune) bool {
			return false
		},
	},
	{
		is: tokenSymbol,
		test: func(r rune) bool {
			switch r {
			default:
				return false
			case '/', '.', '&', '(', ')', '|', ',', '*', '-', '+', '=', ';', '@':
				return true
			}
		},
		while: func(r rune) bool {
			return false
		},
	},
	{
		is: tokenNumber,
		test: func(r rune) bool {
			return unicode.IsNumber(r)
		},
		while: func(r rune) bool {
			return unicode.IsNumber(r) || r == '.' || r == '_'
		},
	},
	{
		is: tokenQuote,
		test: func(r rune) bool {
			return r == '"'
		},
		end: []lexRune{
			func(r rune) bool {
				return r == '"'
			},
		},
	},
}

func (s *state) runLexerItem(c lex, r rune, size int) (bool, error) {
	if err := s.ctx.Err(); err != nil {
		return false, err
	}
	if !c.test(r) {
		return false, nil
	}
	if len(c.next) > 0 {
		r, size, err := s.buf.ReadRune()
		if err != nil {
			return false, err
		}
		s.nextBuf = append(s.nextBuf, r)
		for _, subc := range c.next {
			ok, err := s.runLexerItem(subc, r, size)
			if err != nil {
				return false, err
			}
			if ok {
				return true, nil
			}
		}
		err = s.buf.UnreadRune()
		if err != nil {
			return false, err
		}
		s.nextBuf = s.nextBuf[:len(s.nextBuf)-1]
	}
	if c.while == nil && len(c.end) == 0 {
		return false, nil
	}
	for _, x := range s.nextBuf {
		s.out.WriteRune(x)
	}
	if c.while != nil {
		for {
			r, size, err := s.buf.ReadRune()
			if err != nil {
				return false, err
			}
			_ = size
			if c.while(r) {
				s.out.WriteRune(r)
				continue
			}
			err = s.buf.UnreadRune()
			if err != nil {
				return false, err
			}
			s.emit(c.is)
			return true, nil
		}
	}
	if len(c.end) > 0 {
		buf := make([]rune, len(c.end))
		endIndex := 0
		for {
			r, size, err := s.buf.ReadRune()
			if err != nil {
				return false, err
			}
			_ = size
			if !c.end[endIndex](r) {
				for i := 0; i < endIndex; i++ {
					s.out.WriteRune(buf[i])
				}
				s.out.WriteRune(r)
				endIndex = 0
				continue
			}
			buf[endIndex] = r
			endIndex++
			if len(c.end) <= endIndex {
				if c.excludeEnd {
					for i := 0; i < len(c.end); i++ {
						err = s.buf.UnreadRune()
						if err != nil {
							return false, err
						}
					}
				} else {
					for i := 0; i < endIndex; i++ {
						s.out.WriteRune(buf[i])
					}
				}
				s.emit(c.is)
				return true, nil
			}
		}
	}
	return false, nil
}
func (p Pos) add(s string) Pos {
	p.Byte += int64(len(s))
	if n := strings.Count(s, "\n"); n > 0 {
		p.Line += int64(n)
		s = s[strings.LastIndex(s, "\n")+1:]
		p.LineRune = 1
	}
	p.LineRune += int64(utf8.RuneCountInString(s))
	return p
}

func (s *state) runLexer() error {
loop:
	for {
		if err := s.ctx.Err(); err != nil {
			return err
		}
		r, size, err := s.buf.ReadRune()
		if err != nil {
			return err
		}
		s.nextBuf = append(s.nextBuf, r)
		for _, c := range lexRoot {
			ok, err := s.runLexerItem(c, r, size)
			if err != nil {
				return err
			}
			if ok {
				continue loop
			}
		}
		return fmt.Errorf("no state for <%d> %q", r, string(r))
	}
}

func (s *state) emit(t tokenType) {
	if err := s.ctx.Err(); err != nil {
		return
	}
	v := s.out.String()

	end := s.pos.add(v)
	s.emitToken(lexToken{Start: s.pos, End: end, Type: t, Value: v})

	s.pos = end
	s.out.Reset()
	s.nextBuf = s.nextBuf[:0]
}

type lexToken struct {
	Start    Pos
	End      Pos
	Type     tokenType
	Value    string
	Comments *parseCommentBlock
}

func (s *state) emitToken(t lexToken) {
	if t.Type == tokenWhitespace {
		return
	}
	// For now, remove comments as well.
	// These comments are not currently used. Possibly remove fully again.
	if t.Type == tokenComment {
		if s.comments == nil {
			s.comments = &parseCommentBlock{}
		}
		c := &parseCommentLine{
			Comment: t.Value,
			Start:   t.Start,
			End:     t.End,
			Suffix:  s.prevEmit.Type != tokenNewline,
		}
		if c.Suffix {
			s.comments.Suffix = append(s.comments.Suffix, c)
		} else {
			s.comments.Before = append(s.comments.Before, c)
		}
		return
	}
	if t.Type == tokenNewline {
		t = lexToken{Start: t.Start, Type: tokenEOS}
	}
	if t.Type == tokenSymbol && t.Value == ";" {
		t = lexToken{Start: t.Start, Type: tokenEOS}
	}
	prev := s.prevEmit
	if s.comments != nil {
		if len(s.comments.Suffix) > 0 {
			prev.Comments = s.comments
		} else {
			t.Comments = s.comments
		}
		s.comments = nil
	}
	if prev.Type != tokenUnknown {
		s.emitter <- prev
	}
	s.prevEmit = t
	if t.Type == tokenEOF {
		s.emitter <- t
		s.prevEmit = lexToken{}
	}
}

type tokenError struct {
	msg   string
	token lexToken
}

func (err *tokenError) Error() string {
	t := err.token
	return fmt.Sprintf("%s:%d:%d %s <%s %q>", t.Start.File.Name, t.Start.Line, t.Start.LineRune, err.msg, t.Type, t.Value)
}

func terr(msg string, t lexToken) error {
	return &tokenError{msg: msg, token: t}
}

//go:generate stringer -trimprefix statement -type statementType

type statementType int

const (
	statementUnknown statementType = iota
	statementComment
	statementContext
	statementCreate
	statementSet
	statementSchema
	statementVar
)

type parsePart interface {
	AssignNext(t lexToken) (usedToken bool, next parsePart, err error)
	WriteToBuilder(buf *strings.Builder, level int)
	// Child returns the parsePart at index, as well as the relative
	// path the child is at.
	Child(index int) (path []string, part parsePart)
}

type parseRoot struct {
	Statements []*parseStatement
}

type parseStatement struct {
	Type       statementType
	Identifier *parseFullIdentifier
	Value      *parseValueList
	Comments   *parseCommentBlock
	Start      Pos
	End        Pos
}

type parseFullIdentifier struct {
	Parts []lexToken
	Start Pos
	End   Pos
}

func (p *parseFullIdentifier) String() string {
	buf := &strings.Builder{}
	p.WriteToBuilder(buf, 0)
	return buf.String()
}

func (p *parseFullIdentifier) IdentifierParts() []string {
	ret := make([]string, 0, len(p.Parts)+1)
	for _, token := range p.Parts {
		switch token.Type {
		case tokenIdentifier:
			ret = append(ret, token.Value)
		}
	}
	return ret
}

/*
	Table = []Row
	Row = []Cell
	Cell = []Value
	Value = (Token | Table)
*/

type parseComplexItem struct {
	Token lexToken
	Table *parseTableValue
}

type parseValueList struct {
	Values []*parseComplexItem
}

func (pvl *parseValueList) String() string {
	if pvl == nil {
		return ""
	}
	if len(pvl.Values) == 0 {
		return ""
	}
	v0 := pvl.Values[0]
	return v0.Token.Value
}

type parseRow struct {
	Cells []*parseValueList
	Start Pos
	End   Pos
}
type parseTableValue struct {
	Rows  []*parseRow
	Start Pos
	End   Pos
}
type parseCommentLine struct {
	Comment string
	Suffix  bool // Not a whole line.
	Start   Pos
	End     Pos
}

type parseCommentBlock struct {
	Before []*parseCommentLine
	Suffix []*parseCommentLine
}

func (p *parseRoot) WriteToBuilder(buf *strings.Builder, level int) {
	for _, st := range p.Statements {
		buf.WriteString("Statement: ")
		st.WriteToBuilder(buf, level)
		buf.WriteRune('\n')
	}
}
func (p *parseRoot) String() string {
	buf := &strings.Builder{}
	p.WriteToBuilder(buf, 0)
	return buf.String()
}

func (p *parseRoot) AssignNext(t lexToken) (bool, parsePart, error) {
	switch t.Type {
	default:
		return false, nil, terr("unknown root state", t)
	case tokenEOS:
		return true, p, nil
	case tokenIdentifier:
		st := &parseStatement{
			Start: t.Start,
			End:   t.End,
		}
		p.Statements = append(p.Statements, st)

		return false, st, nil
	}
}

func (p *parseFullIdentifier) AssignNext(t lexToken) (bool, parsePart, error) {
	switch t.Type {
	default:
		return false, nil, terr("unknown full identifier token type", t)
	case tokenEOS:
		return false, nil, nil
	case tokenIdentifier, tokenSymbol:
		if t.Value == ";" {
			return true, nil, nil
		}
		if len(p.Parts) == 0 {
			p.Parts = append(p.Parts, t)
			return true, p, nil
		}
		prev := p.Parts[len(p.Parts)-1]
		if t.Type == tokenIdentifier && prev.Type == tokenIdentifier {
			p.End = prev.End
			return false, nil, nil
		}
		p.Parts = append(p.Parts, t)
		return true, p, nil
	}
}
func (p *parseFullIdentifier) WriteToBuilder(buf *strings.Builder, level int) {
	for _, v := range p.Parts {
		buf.WriteString(v.Value)
	}
}

func (p *parseValueList) AssignNext(t lexToken) (bool, parsePart, error) {
	switch t.Type {
	default:
		return false, nil, terr("unexpected token type", t)
	case tokenEOS:
		return false, nil, nil
	case tokenSymbol:
		switch t.Value {
		default:
			p.Values = append(p.Values, &parseComplexItem{Token: t})
			return true, p, nil
		case "|", ",": // Done reading value from table.
			return false, nil, nil
		case ")":
			return false, nil, nil
		case "(":
			if len(p.Values) == 0 {
				return false, nil, terr("must declare type before table or list", t)
			}
			table := &parseTableValue{
				Start: t.Start,
				End:   t.End,
			}
			v := &parseComplexItem{
				Table: table,
			}
			p.Values = append(p.Values, v)
			return true, table, nil
		}
	case tokenIdentifier, tokenNumber, tokenQuote:
		p.Values = append(p.Values, &parseComplexItem{Token: t})
		return true, p, nil
	}
}
func (p *parseValueList) WriteToBuilder(buf *strings.Builder, level int) {
	var prev lexToken
	for i, v := range p.Values {
		if i > 0 {
			if prev.Type != tokenSymbol && !(v.Token.Type == tokenSymbol && v.Token.Value == ".") {
				buf.WriteString(" ")
			}
		}
		if v.Token.Type != tokenUnknown {
			buf.WriteString(v.Token.Value)
		}
		if v.Table != nil {
			v.Table.WriteToBuilder(buf, level+1)
		}
		prev = v.Token
	}
}

func (p *parseRow) AssignNext(t lexToken) (bool, parsePart, error) {
	switch t.Type {
	default:
		if len(p.Cells) == 0 {
			v := &parseValueList{}
			p.Cells = append(p.Cells, v)
			return false, v, nil
		}
		v := p.Cells[len(p.Cells)-1]
		return false, v, nil
	case tokenEOS:
		return false, nil, nil
	case tokenSymbol:
		switch t.Value {
		default:
			if len(p.Cells) == 0 {
				v := &parseValueList{}
				p.Cells = append(p.Cells, v)
				return false, v, nil
			}
			v := p.Cells[len(p.Cells)-1]
			return false, v, nil
		case "|", ",": // Done reading value from table.
			v := &parseValueList{}
			p.Cells = append(p.Cells, v)
			return true, v, nil
		case ")":
			return false, nil, nil
		}
	}
}

func (p *parseRow) WriteToBuilder(buf *strings.Builder, level int) {
	for i, v := range p.Cells {
		if i > 0 {
			buf.WriteString(" | ")
		}
		v.WriteToBuilder(buf, level)
	}
}

func (p *parseTableValue) AssignNext(t lexToken) (bool, parsePart, error) {
	var row *parseRow
	if len(p.Rows) > 0 {
		row = p.Rows[len(p.Rows)-1]
	}

	switch t.Type {
	default:
		if row == nil {
			row = &parseRow{}
			p.Rows = append(p.Rows, row)
		}
		return false, row, nil
	case tokenSymbol:
		switch t.Value {
		default:
			if row == nil {
				row = &parseRow{}
				p.Rows = append(p.Rows, row)
			}
			return false, row, nil
		case ")":
			// Remove trailing empty row added by EOS.
			if row != nil && len(row.Cells) == 0 {
				p.Rows = p.Rows[:len(p.Rows)-1]
			}
			p.End = t.End
			return true, nil, nil
		case "|", ",":
			if row == nil {
				return false, nil, terr(`unexpected "|" before first row`, t)
			}
			v := &parseValueList{}
			row.Cells = append(row.Cells, v)
			return true, row, nil
		}
	case tokenEOS:
		row = &parseRow{}
		p.Rows = append(p.Rows, row)
		return true, row, nil
	}
}
func (p *parseTableValue) WriteToBuilder(buf *strings.Builder, level int) {
	for _, row := range p.Rows {
		buf.WriteRune('\n')
		for i := 0; i < level; i++ {
			buf.WriteRune('\t')
		}
		buf.WriteString("Row: ")
		row.WriteToBuilder(buf, level)
	}
}

func (p *parseStatement) AssignNext(t lexToken) (bool, parsePart, error) {
	if p.Type == statementUnknown {
		switch t.Type {
		default:
			return false, nil, terr("unknown statement token type", t)
		case tokenComment:
			p.Type = statementComment
			p.Comments = t.Comments
			return true, p, nil
		case tokenIdentifier:
			switch t.Value {
			default:
				return false, nil, terr("unknown statment type", t)
			case "context":
				p.Type = statementContext
			case "create":
				p.Type = statementCreate
			case "set":
				p.Type = statementSet
			case "schema":
				p.Type = statementSchema
			case "var":
				p.Type = statementVar
			}
			p.Identifier = &parseFullIdentifier{
				Start: t.Start,
				End:   t.End,
			}
			return true, p.Identifier, nil
		}
	}
	if p.Identifier == nil {
		return false, nil, terr("unexpected state, expected statement Identifier to be set", t)
	}
	if p.Value == nil {
		p.Value = &parseValueList{}
		return false, p.Value, nil
	}
	return false, nil, nil
}
func (p *parseStatement) WriteToBuilder(buf *strings.Builder, level int) {
	buf.WriteString(p.Type.String())
	switch p.Type {
	case statementComment:
		buf.WriteString(fmt.Sprintf("%v", p.Comments))
	default:
		buf.WriteString(" ")
		p.Identifier.WriteToBuilder(buf, level)
		switch {
		case p.Value != nil && len(p.Value.Values) > 0:
			buf.WriteString(" ")
			p.Value.WriteToBuilder(buf, level)
		}
	}
}

func (s *state) runParse(root *parseRoot) error {
fileloop:
	for {
		stack := []parsePart{root}

		t, ok := <-s.emitter
		if !ok {
			return nil
		}

		for {
			next := stack[len(stack)-1]
			usedToken, goNext, err := next.AssignNext(t)
			if err != nil {
				return err
			}
			switch {
			case goNext == next:
				// Nothing.
			case goNext == nil:
				stack = stack[:len(stack)-1]
			default:
				stack = append(stack, goNext)
			}
			if usedToken {
				t, ok = <-s.emitter
				if t.Type == tokenEOF || t.Type == tokenUnknown {
					continue fileloop
				}
			}
		}
	}
}

func (p *parseRoot) Child(index int) (path []string, part parsePart) {
	if index < 0 {
		return nil, nil
	}
	if index >= len(p.Statements) {
		return nil, nil
	}
	st := p.Statements[index]
	return st.Identifier.IdentifierParts(), st.Value
}

func (p *parseFullIdentifier) Child(index int) (path []string, part parsePart) {
	// Identifier has no children.
	return nil, nil
}

func (p *parseValueList) Child(index int) (path []string, part parsePart) {
	if index < 0 {
		return nil, nil
	}
	if index >= len(p.Values) {
		return nil, nil
	}
	return nil, nil // p.Values[index]
}

func (p *parseRow) Child(index int) (path []string, part parsePart) {
	if index < 0 {
		return nil, nil
	}
	if index >= len(p.Cells) {
		return nil, nil
	}
	return nil, p.Cells[index]
}

func (p *parseTableValue) Child(index int) (path []string, part parsePart) {
	if index < 0 {
		return nil, nil
	}
	if index >= len(p.Rows) {
		return nil, nil
	}
	return nil, p.Rows[index]
}

func (p *parseStatement) Child(index int) (path []string, part parsePart) {
	if index != 0 {
		return nil, nil
	}
	return nil, p.Value
}
