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

type Selector struct {
	Raw string
}

type Context struct {
	Selector Selector
}
type Create struct {
	Selector Selector
}
type Set struct {
	Selector Selector
	Value    Value
}

type Value struct {
	ValueName string
	Array     *ValueArray
	Table     *ValueTable
}

type ValueArray struct {
	Raw []string
}
type ValueTable struct {
	RawHeader []string
	RawValues [][]string
}
type file struct {
	Name string

	Tokens []lexToken
}
type pos struct {
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
	pos pos

	nextBuf []rune
	out     *strings.Builder

	prevTokenSymbol bool
}

func (s *state) load(name string, r io.Reader) error {
	f := &file{Name: name}
	s.files = append(s.files, f)
	s.buf = bufio.NewReader(r)
	s.out = &strings.Builder{}
	s.pos = pos{
		File:     f,
		Line:     1,
		LineRune: 1,
		Byte:     0,
	}

	err := s.runLexer()
	if err == io.EOF {
		s.emitToken(lexToken{Pos: s.pos, Type: tokenEOF})
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
		ctx:  ctx,
		stop: cancel,
	}
	defer cancel()

	err := fr.Load(s.load)
	if err != nil {
		return nil, err
	}
	return s.finalize()
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
			case '/', '.', '&', '(', ')', '|', ',', '*', '-', '+', '=', ';':
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
func (p pos) add(s string) pos {
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

	s.emitToken(lexToken{Pos: s.pos, Type: t, Value: v})

	s.pos = s.pos.add(v)
	s.out.Reset()
	s.nextBuf = s.nextBuf[:0]
}

type lexToken struct {
	Pos   pos
	Type  tokenType
	Value string
}

func (s *state) emitToken(token lexToken) {
	token.Pos.File.Tokens = append(token.Pos.File.Tokens, token)
}

func (s *state) finalize() (*parseRoot, error) {
	root := &parseRoot{}
	for _, f := range s.files {
		err := finalizeFile(f, root)
		if err != nil {
			return root, err
		}
	}
	return root, nil
}

/*
type parseState int

const (
	parseStateUnknown parseState = iota
	parseStateRoot
	parseStateContext
	parseStateSet
	parseStateArray
	parseStateTable
)
*/

type tokenError struct {
	msg   string
	token lexToken
}

func (err *tokenError) Error() string {
	t := err.token
	return fmt.Sprintf("%s:%d:%d <%s %q> %s", t.Pos.File.Name, t.Pos.Line, t.Pos.LineRune, t.Type, t.Value, err.msg)
}

func terr(msg string, t lexToken) error {
	return &tokenError{msg: msg, token: t}
}

//go:generate stringer -trimprefix statement -type statementType

type statementType int

const (
	statementUnknown statementType = iota
	statementContext
	statementCreate
	statementSet
)

type parsePart interface {
	AssignNext(t lexToken) (usedToken bool, next parsePart, err error)
	WriteToBuilder(buf *strings.Builder)
}

type parseRoot struct {
	Statements []*parseStatement
}

func (p *parseRoot) WriteToBuilder(buf *strings.Builder) {
	for _, st := range p.Statements {
		buf.WriteString("Statement: ")
		st.WriteToBuilder(buf)
		buf.WriteRune('\n')
	}
}
func (p *parseRoot) String() string {
	buf := &strings.Builder{}
	p.WriteToBuilder(buf)
	return buf.String()
}

func (p *parseRoot) AssignNext(t lexToken) (bool, parsePart, error) {
	switch t.Type {
	default:
		return false, nil, terr("unknown root state", t)
	case tokenEOS:
		return true, p, nil
	case tokenIdentifier:
		st := &parseStatement{}
		p.Statements = append(p.Statements, st)

		return false, st, nil
	}
}

type parseFullIdentifier struct {
	Parts []lexToken
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
			return false, nil, nil
		}
		p.Parts = append(p.Parts, t)
		return true, p, nil
	}
}
func (p *parseFullIdentifier) WriteToBuilder(buf *strings.Builder) {
	for _, v := range p.Parts {
		buf.WriteString(v.Value)
	}
}

type parseComplexItem struct {
	Token   lexToken
	Complex *parseComplexValue
}

type parseComplexValue struct {
	Values []*parseComplexItem
	List   *parseListValue
	Table  *parseTableValue
}

func (p *parseComplexValue) AssignNext(t lexToken) (bool, parsePart, error) {
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
		case "|": // Done reading value from table.
			return true, nil, nil
		case ")":
			return false, nil, nil
		case "(":
			if len(p.Values) == 0 {
				return false, nil, terr("must declare type before table or list", t)
			}
			v := p.Values[len(p.Values)-1]
			table := &parseTableValue{}
			v.Complex = &parseComplexValue{
				Table: table,
			}
			// buf := &strings.Builder{}
			// p.WriteToBuilder(buf)
			// fmt.Println(buf)
			return true, table, nil
		}
	case tokenIdentifier, tokenNumber:
		p.Values = append(p.Values, &parseComplexItem{Token: t})
		return true, p, nil
	}
}
func (p *parseComplexValue) WriteToBuilder(buf *strings.Builder) {
	for i, v := range p.Values {
		if i > 0 {
			buf.WriteRune(' ')
		}
		if v.Token.Type != tokenUnknown {
			buf.WriteString(v.Token.Value)
		}
		if v.Complex != nil {
			v.Complex.WriteToBuilder(buf)
		}
	}
	if p.List != nil {
		buf.WriteString(", List: ")
		p.List.WriteToBuilder(buf)
	}
	if p.Table != nil {
		buf.WriteString(", Table:\n")
		p.Table.WriteToBuilder(buf)
	}
}

type parseListValue struct {
	Items []*parseComplexValue
}

func (p *parseListValue) AssignNext(t lexToken) (bool, parsePart, error) {
	if t.Type == tokenSymbol && t.Value == ")" {
		return false, nil, nil
	}
	// TODO(daniel.theophanes): Actually store values.
	return true, p, nil
}
func (p *parseListValue) WriteToBuilder(buf *strings.Builder) {
	for i, cell := range p.Items {
		if i > 0 {
			buf.WriteString(", ")
		}
		cell.WriteToBuilder(buf)
	}
}

type parseTableValue struct {
	Rows []*parseComplexValue
}

func (p *parseTableValue) AssignNext(t lexToken) (bool, parsePart, error) {
	var row *parseComplexValue
	if len(p.Rows) > 0 {
		row = p.Rows[len(p.Rows)-1]
	}

	switch t.Type {
	default:
		if row == nil {
			row = &parseComplexValue{}
			p.Rows = append(p.Rows, row)
		}
		return false, row, nil
	case tokenSymbol:
		switch t.Value {
		default:
			if row == nil {
				row = &parseComplexValue{}
				p.Rows = append(p.Rows, row)
			}
			return false, row, nil
		case ")":
			// Remove trailing empty row added by EOS.
			if row != nil && len(row.Values) == 0 && row.List == nil && row.Table == nil {
				p.Rows = p.Rows[:len(p.Rows)-1]
			}
			return true, nil, nil
		case "|":
			if row == nil {
				return false, nil, terr(`unexpected "|" before first row`, t)
			}
			v := &parseComplexItem{}
			row.Values = append(row.Values, v)
			return true, v.Complex, nil
		}
	case tokenEOS:
		row = &parseComplexValue{}
		p.Rows = append(p.Rows, row)
		return true, row, nil
	}
}
func (p *parseTableValue) WriteToBuilder(buf *strings.Builder) {
	for i, row := range p.Rows {
		if i > 0 {
			buf.WriteRune('\n')
		}
		buf.WriteRune('\t')
		buf.WriteString("Row: ")
		row.WriteToBuilder(buf)
	}
}

type parseStatement struct {
	Type       statementType
	Identifier *parseFullIdentifier
	Value      *parseComplexValue
}

func (p *parseStatement) AssignNext(t lexToken) (bool, parsePart, error) {
	if p.Type == statementUnknown {
		switch t.Type {
		default:
			return false, nil, terr("unknown statement token type", t)
		case tokenIdentifier:
			switch t.Value {
			default:
				return false, nil, terr("unknown statment type %s", t)
			case "context":
				p.Type = statementContext
			case "create":
				p.Type = statementCreate
			case "set":
				p.Type = statementSet
			}
			p.Identifier = &parseFullIdentifier{}
			return true, p.Identifier, nil
		}
	}
	if p.Identifier == nil {
		return false, nil, terr("unexpected state, expected statement Identifier to be set", t)
	}
	if p.Value == nil {
		p.Value = &parseComplexValue{}
		return false, p.Value, nil
	}
	return false, nil, nil
}
func (p *parseStatement) WriteToBuilder(buf *strings.Builder) {
	buf.WriteString("Type: ")
	buf.WriteString(p.Type.String())
	buf.WriteString(", Identifier: ")
	p.Identifier.WriteToBuilder(buf)
	switch {
	case p.Value != nil:
		buf.WriteString(", Value: ")
		p.Value.WriteToBuilder(buf)
	}
}

func finalizeFile(f *file, root *parseRoot) error {
	tokenIndex := 0
	// prevTokenSymbol := true
	nextTokenPre := func() (t lexToken) {
		for {
			if tokenIndex >= len(f.Tokens) {
				return t
			}
			t = f.Tokens[tokenIndex]
			tokenIndex++

			if t.Type == tokenWhitespace {
				continue
			}
			// For now, remove comments as well.
			if t.Type == tokenComment {
				continue
			}
			// prevTokenSymbol = t.Type == tokenSymbol || t.Type == tokenNewline
			if t.Type == tokenNewline {
				return lexToken{Pos: t.Pos, Type: tokenEOS}
			}
			if t.Type == tokenSymbol && t.Value == ";" {
				return lexToken{Pos: t.Pos, Type: tokenEOS}
			}
			if t.Type == tokenNewline {
				continue
			}
			return t
		}
	}
	nextToken := func() lexToken {
		t := nextTokenPre()
		// fmt.Printf("%s:%d:%d %s %q\n", t.Pos.File.Name, t.Pos.Line, t.Pos.LineRune, t.Type, t.Value)
		return t
	}

	stack := []parsePart{root}

	t := nextToken()

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
			t = nextToken()
			if t.Type == tokenEOF {
				break
			}
		}
	}

	return nil
}
