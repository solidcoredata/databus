package parse

import (
	"bytes"
	"context"
	"io"
	"io/ioutil"
	"regexp"
	"sort"
	"strings"
	"testing"
)

type mapFileReader struct {
	Lookup map[string]io.Reader
	List   []string
}

// func (fr *mapFileReader) Load(loader FileLoader) error {
// 	names := make([]string, 0, len(fr))
// 	for name := range fr {
// 		names = append(names, name)
// 	}
// 	sort.Strings(names)
// 	for _, name := range names {
// 		r := fr[name]
// 		err := loader(name, r)
// 		if err != nil {
// 			return err
// 		}
// 	}
// 	return nil
// }

func (fr *mapFileReader) filterList(filter string) []string {
	out := []string{}
	for _, item := range fr.List {
		if strings.HasSuffix(item, filter) {
			out = append(out, item)
		}
	}
	return out
}

func loadMapFileReader(path string) (*mapFileReader, error) {
	bb, err := ioutil.ReadFile(path)
	if err != nil {
		return nil, err
	}
	re, err := regexp.Compile(`(?m)^--[[:blank:]]*(?P<filename>[\w/.]+)[[:blank:]]*--$`)
	if err != nil {
		return nil, err
	}

	fr := &mapFileReader{
		Lookup: map[string]io.Reader{},
	}

	indexes := re.FindAllSubmatchIndex(bb, -1)
	for index, x := range indexes {
		end := len(bb)
		if index+1 < len(indexes) {
			next := indexes[index+1]
			end = next[0]
		}
		body := bb[x[1]:end]
		name := string(bb[x[2]:x[3]])
		fr.Lookup[name] = bytes.NewReader(body)
		fr.List = append(fr.List, name)
	}
	sort.Strings(fr.List)
	return fr, nil
}

func TestLibrary(t *testing.T) {
	const from = "testdata/library.scdm"
	fr, err := loadMapFileReader(from)
	if err != nil {
		t.Fatal(err)
	}

	load := func(name string) string {
		r := fr.Lookup[name+".golden"]
		if r == nil {
			return ""
		}
		b, _ := ioutil.ReadAll(r)
		return strings.TrimSpace(string(b))
	}

	ctx := context.Background()
	for _, name := range fr.filterList(".scd") {
		t.Run(name, func(t *testing.T) {
			list, err := ParseFile(ctx, name, fr.Lookup[name])
			if list == nil && err != nil {
				t.Fatal(err)
			}
			got := strings.TrimSpace(list.String())
			want := load(name)
			if got != want {
				t.Logf("got\n%v\n; want\n%v\n", got, want)
				t.Log("======")
				gg, ww := strings.Split(got, "\n"), strings.Split(want, "\n")
				min := len(gg)
				if len(ww) < min {
					min = len(ww)
				}
				if len(ww) != len(gg) {
					t.Logf("got %d lines; want %d lines", len(gg), len(ww))
				}
				for i := 0; i < min; i++ {
					g, w := gg[i], ww[i]
					if g == w {
						t.Log(w)
					} else {
						t.Log("-", w)
						t.Log("+", g)
					}
				}
				if err != nil {
					t.Fatal(err)
				}
				t.Fatal("incorrect line")
			}
		})
	}
}
