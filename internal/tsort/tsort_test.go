package tsort

import (
	"reflect"
	"testing"
)

type tcollection struct {
	nodes []tnode
}

func (c tcollection) Index(i int) Node {
	return c.nodes[i]
}

func (c tcollection) Len() int {
	return len(c.nodes)
}

func (c tcollection) Swap(i, j int) {
	c.nodes[i], c.nodes[j] = c.nodes[j], c.nodes[i]
}

type tnode struct {
	id string
	d  []string
}

func (n tnode) ID() string {
	return n.id
}
func (n tnode) ToNode() []string {
	return n.d
}

func TestTSort(t *testing.T) {
	list := []struct {
		Name string
		In   []tnode
		Out  []string
		Err  string
	}{
		{
			Name: "A",
			In: []tnode{
				{id: "a", d: []string{"b", "c"}},
				{id: "c", d: []string{"d"}},
				{id: "b", d: []string{"d"}},
				{id: "d"},
			},
			Out: []string{"d", "b", "c", "a"},
		},
		{
			Name: "B",
			In: []tnode{
				{id: "a", d: []string{"b", "c"}},
				{id: "b", d: []string{"d"}},
				{id: "c", d: []string{"d"}},
				{id: "d", d: []string{"c"}},
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
			err := Sort(tcollection{nodes: item.In})
			if len(item.Err) > 0 {
				if err == nil {
					t.Fatalf("got <nil> error, expected: %v", item.Err)
				}
				if item.Err != err.Error() {
					t.Fatalf("expected error %v, got error %v", item.Err, err)
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			l := make([]string, len(item.In))
			for i, n := range item.In {
				l[i] = n.id
			}
			if !reflect.DeepEqual(l, item.Out) {
				t.Fatalf("expected layers %v, got layers %v", item.Out, l)
			}
		})
	}
}
