package main

import (
    "io"
)
type FileReader map[string]io.Reader

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
    Value Value
}

type Value struct {
    ValueName string
    Array *ValueArray
    Table *ValueTable
}

type ValueArray struct {
    Raw []string
}
type ValueTable struct {
    RawHeader []string
    RawValues [][]string
}

func parse(fr FileReader) error {
    return nil
}
