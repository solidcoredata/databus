package parse

import (
	"bufio"
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
	name string
}
type pos struct {
	File     *file
	Line     int64 // line in input (starting at 1)
	LineRune int64 // rune in line (starting at 1)
	Byte     int64 // byte in input (starting at 0)
}
type state struct {
	buf *bufio.Reader
	pos pos

	nextBuf []rune
	out     *strings.Builder
}
type token struct {
	file  *file
	pos   pos
	value []byte
}

func (s *state) load(name string, r io.Reader) error {
	s.buf = bufio.NewReader(r)
	s.out = &strings.Builder{}
	s.pos = pos{
		File:     &file{name: name},
		Line:     1,
		LineRune: 1,
		Byte:     0,
	}

	err := s.runLexer()
	if err == io.EOF {
		return nil
	}
	return err
}
func (s *state) finalize() error {
	return nil
}

func ParseFile(fr FileReader) (*state, error) {
	s := &state{}
	err := fr.Load(s.load)
	if err != nil {
		return nil, err
	}
	return s, s.finalize()
}

type parseState int

const (
	parseStateUnknown parseState = iota
	parseStateRoot
	parseStateContext
	parseStateSet
	parseStateArray
	parseStateTable
)

type parse struct {
}

var parseRoot = &parse{}

type lexRune func(r rune) bool

type lex struct {
	is    tokenType
	test  lexRune
	next  []lex
	while lexRune
	end   []lexRune
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
		endIndex := 0
		for {
			r, size, err := s.buf.ReadRune()
			if err != nil {
				return false, err
			}
			_ = size
			s.out.WriteRune(r)
			if !c.end[endIndex](r) {
				endIndex = 0
				continue
			}
			endIndex++
			if len(c.end) <= endIndex {
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

func (s *state) emit(t tokenType) {
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
	fmt.Printf("token@%v: <%s> %q\n", token.Pos, token.Type, token.Value)
}

func (s *state) runLexer() error {
loop:
	for {
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
