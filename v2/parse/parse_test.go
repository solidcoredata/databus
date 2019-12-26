package parse

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"io/ioutil"
	"regexp"
	"testing"
)

type mapFileReader map[string]io.Reader

func (fr mapFileReader) Load(loader FileLoader) error {
	for name, r := range fr {
		err := loader(name, r)
		if err != nil {
			return err
		}
	}
	return nil
}

func loadMapFileReader(path string) (mapFileReader, error) {
	bb, err := ioutil.ReadFile(path)
	if err != nil {
		return nil, err
	}
	re, err := regexp.Compile(`(?m)^--[[:blank:]]*(?P<filename>[\w/.]+)[[:blank:]]*--$`)
	if err != nil {
		return nil, err
	}

	fr := mapFileReader{}

	indexes := re.FindAllSubmatchIndex(bb, -1)
	for index, x := range indexes {
		end := len(bb)
		if index+1 < len(indexes) {
			next := indexes[index+1]
			end = next[0]
		}
		body := bb[x[1]:end]
		name := string(bb[x[2]:x[3]])
		fr[name] = bytes.NewReader(body)
	}
	return fr, nil
}

func TestLibrary(t *testing.T) {
	const from = "testdata/library.scdm"
	fr, err := loadMapFileReader(from)
	if err != nil {
		t.Fatal(err)
	}

	ctx := context.Background()
	st, err := ParseFile(ctx, fr)
	if err != nil {
		t.Fatal(err)
	}
	fmt.Printf("state: %+v\n", *st)
}
