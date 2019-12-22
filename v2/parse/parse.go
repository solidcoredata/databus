package parse

import (
	"io"
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

type state struct {
}

func (s *state) load(name string, r io.Reader) error {
	return nil
}
func (s *state) finalize() error {
	return nil
}
func parse(fr FileReader) (*state, error) {
	s := &state{}
	err := fr.Load(s.load)
	if err != nil {
		return nil, err
	}
	return s, s.finalize()
}
