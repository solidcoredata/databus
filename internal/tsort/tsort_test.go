package tsort

import (
	"reflect"
	"testing"
)

func TestTSort(t *testing.T) {
	list := []struct {
		Name string
		In   []Node
		Out  [][]string
		Err  string
	}{
		{
			Name: "A",
			In: []Node{
				{ID: "a", Dependencies: []string{"b", "c"}},
				{ID: "c", Dependencies: []string{"d"}},
				{ID: "b", Dependencies: []string{"d"}},
				{ID: "d"},
			},
			Out: [][]string{{"d"}, {"b", "c"}, {"a"}},
		},
		{
			Name: "B",
			In: []Node{
				{ID: "a", Dependencies: []string{"b", "c"}},
				{ID: "b", Dependencies: []string{"d"}},
				{ID: "c", Dependencies: []string{"d"}},
				{ID: "d", Dependencies: []string{"c"}},
			},
			Err: `circular reference:
- a
--- b
--- c
- b
--- d
- c
--- d
- d
--- c
`,
		},
	}
	for _, item := range list {
		t.Run(item.Name, func(t *testing.T) {
			l, err := Sort(item.In)
			if (len(item.Err) == 0 && err != nil) || (len(item.Err) > 0 && err == nil) {
				t.Fatalf("expected error %v, got error %v", item.Err, err)
			}
			if !reflect.DeepEqual(l, item.Out) {
				t.Fatalf("expected layers %v, got layers %v", item.Out, l)
			}
		})
	}
}
