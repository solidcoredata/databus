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

func (p Pos) String() string {
	return fmt.Sprintf("%s:%d%d", p.File.Name, p.Line, p.LineRune)
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

	prevEmit lexToken
}

func (s *state) load(name string, r io.Reader) (parseLineList, error) {
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

	le := &lineEmitter{
		Root: &parseLine{
			Group: groupStruct,
		},
	}
	var previousToken lexToken
	for {
		lt, err := s.runLexer()
		if err == io.EOF {
			lt = lexToken{Start: s.pos, Type: tokenEOF}
			err = nil
		}
		if err != nil {
			return nil, fmt.Errorf("%s:%d:%d %v", s.pos.File.Name, s.pos.Line, s.pos.LineRune, err)
		}
		if lt.Type == tokenWhitespace {
			continue
		}
		if lt.Type == tokenSymbol && lt.Value == ";" {
			lt = lexToken{Start: s.pos, Type: tokenEOS}
		}
		if lt.Type == tokenNewline {
			if previousToken.Type == tokenSymbol {
				continue
			}
			lt = lexToken{Start: s.pos, Type: tokenEOS}
		}
		switch previousToken.Type {
		case tokenEOS, tokenUnknown:
			switch lt.Type {
			case tokenEOS:
				continue
			}
		}
		err = le.EmitToken(lt)
		if err != nil {
			if err == io.EOF {
				return le.all, nil
			}
			return nil, err
		}
		previousToken = lt
	}
}

type parseLineList []*parseLine

func (list parseLineList) String() string {
	sb := &strings.Builder{}
	for i, line := range list {
		if i != 0 {
			sb.WriteRune('\n')
		}
		sb.WriteString(line.String())
	}
	return sb.String()
}

func ParseFile(ctx context.Context, fileName string, r io.Reader) (parseLineList, error) {
	ctx, cancel := context.WithCancel(ctx)
	s := &state{
		ctx:  ctx,
		stop: cancel,
	}

	return s.load(fileName, r)
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
			return unicode.IsLetter(r) || r == '_' || r == '@'
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
			case '.', '(', ')', '{', '}', '|', ',', ';':
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

func (s *state) runLexerItem(c lex, r rune, size int) (lexToken, error) {
	lt := lexToken{}
	if err := s.ctx.Err(); err != nil {
		return lt, err
	}
	if !c.test(r) {
		return lt, nil
	}
	if len(c.next) > 0 {
		r, size, err := s.buf.ReadRune()
		if err != nil {
			return lt, err
		}
		s.nextBuf = append(s.nextBuf, r)
		for _, subc := range c.next {
			lt, err := s.runLexerItem(subc, r, size)
			if err != nil {
				return lt, err
			}
			if lt.Type != tokenUnknown {
				return lt, nil
			}
		}
		err = s.buf.UnreadRune()
		if err != nil {
			return lt, err
		}
		s.nextBuf = s.nextBuf[:len(s.nextBuf)-1]
	}
	if c.while == nil && len(c.end) == 0 {
		return lt, nil
	}
	for _, x := range s.nextBuf {
		s.out.WriteRune(x)
	}
	if c.while != nil {
		for {
			r, size, err := s.buf.ReadRune()
			if err != nil {
				return lt, err
			}
			_ = size
			if c.while(r) {
				s.out.WriteRune(r)
				continue
			}
			err = s.buf.UnreadRune()
			if err != nil {
				return lt, err
			}

			return s.emit(c.is), nil
		}
	}
	if len(c.end) > 0 {
		buf := make([]rune, len(c.end))
		endIndex := 0
		for {
			r, size, err := s.buf.ReadRune()
			if err != nil {
				return lt, err
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
							return lt, err
						}
					}
				} else {
					for i := 0; i < endIndex; i++ {
						s.out.WriteRune(buf[i])
					}
				}

				return s.emit(c.is), nil
			}
		}
	}
	return lt, nil
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

func (s *state) runLexer() (lexToken, error) {
	lt := lexToken{}
	r, size, err := s.buf.ReadRune()
	if err != nil {
		return lt, err
	}
	s.nextBuf = append(s.nextBuf, r)
	for _, c := range lexRoot {
		lt, err = s.runLexerItem(c, r, size)
		if err != nil {
			return lt, err
		}
		if lt.Type != tokenUnknown {
			return lt, nil
		}
	}
	return lt, fmt.Errorf("no state for <%d> %q", r, string(r))
}

func (s *state) emit(t tokenType) lexToken {
	v := s.out.String()

	end := s.pos.add(v)
	lt := lexToken{Start: s.pos, End: end, Type: t, Value: v}

	s.pos = end
	s.out.Reset()
	s.nextBuf = s.nextBuf[:0]
	return lt
}

type lexToken struct {
	Start Pos
	End   Pos
	Type  tokenType
	Value string
}

func (lt lexToken) String() string {
	return fmt.Sprintf("%s:%d:%d-%d:%d (%v) %q", lt.Start.File.Name, lt.Start.Line, lt.Start.LineRune, lt.End.Line, lt.End.LineRune, lt.Type, lt.Value)
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
