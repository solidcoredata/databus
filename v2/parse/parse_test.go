package parse

import (
	"bytes"
	"context"
	"io"
	"io/ioutil"
	"regexp"
	"sort"
	"testing"
)

type mapFileReader map[string]io.Reader

func (fr mapFileReader) Load(loader FileLoader) error {
	names := make([]string, 0, len(fr))
	for name := range fr {
		names = append(names, name)
	}
	sort.Strings(names)
	for _, name := range names {
		r := fr[name]
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
	root, err := ParseFile(ctx, fr)
	if root != nil {
		t.Logf("root: %s\n", root)
	}
	if err != nil {
		t.Fatal(err)
	}
	_, err = Parse2(root)
	if err != nil {
		t.Fatal(err)
	}
}
