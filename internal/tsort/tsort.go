// Package tsort implements a layered topological sort.
package tsort

import (
	"fmt"
	"sort"
	"strings"
)

// NodeCollection contains a list of nodes that can be sorted.
type NodeCollection interface {
	Index(i int) Node
	Len() int
	Swap(i, j int)
}

type sortableNodeCollection struct {
	NodeCollection

	pos map[string]int
}

func (s sortableNodeCollection) Less(i, j int) bool {
	a, b := s.Index(i), s.Index(j)
	an, bn := a.ID(), b.ID()
	ai, bi := s.pos[an], s.pos[bn]
	return ai < bi
}

// Node that may contain zero or more dependencies.
type Node interface {
	ID() string
	ToNode() []string
}

// ErrCircular is may be returned from Sort if the given nodes
// form a cyclic graph.
type ErrCircular struct {
	remaining namedSet
}

func (err ErrCircular) Error() string {
	return err.remaining.String()
}

type namedSet map[string]map[string]bool

func (nset namedSet) String() string {
	// Copy maps to list and sort list before displaying.
	// Ensures a consistent output.
	buf := &strings.Builder{}
	buf.WriteString("circular reference:\n")
	list := make([]string, 0, len(nset))
	setList := []string{}
	for key := range nset {
		list = append(list, key)
	}
	sort.Strings(list)
	for _, key := range list {
		set := nset[key]
		buf.WriteString("- ")
		buf.WriteString(key)
		buf.WriteRune('\n')

		setList = setList[:0]
		for dep := range set {
			setList = append(setList, dep)
		}
		sort.Strings(setList)
		for _, dep := range setList {
			buf.WriteString("--- ")
			buf.WriteString(dep)
			buf.WriteRune('\n')
		}
	}
	return buf.String()
}

// Sort the given nodes into layers.
func Sort(nc NodeCollection) (err error) {
	IDDep := make(namedSet)
	DepID := make(namedSet)
	for i := 0; i < nc.Len(); i++ {
		n := nc.Index(i)

		nid := n.ID()
		if _, ok := IDDep[nid]; ok {
			return fmt.Errorf("Node %q is present more then once", nid)
		}
		IDDep[nid] = make(map[string]bool)
		for _, dep := range n.ToNode() {
			if dep == nid {
				continue
			}
			IDDep[nid][dep] = true
			if _, ok := DepID[dep]; !ok {
				DepID[dep] = make(map[string]bool)
			}
			DepID[dep][nid] = true
		}
	}

	type pair struct {
		ID, Dep string
	}
	layers := make([][]string, 0, nc.Len())

	// Process nodes until all nodes have no dependencies.
	// If one or more node remains, then there is a circular reference.
	for len(IDDep) > 0 {
		loopID := make([]string, 0)
		for k, v := range IDDep {
			if 0 == len(v) {
				loopID = append(loopID, k)
			}
		}

		// Remove unused dependencies before checking it again.
		// If there are no additional dependencies remove it.
		if 0 == len(loopID) {
			remove := []pair{}
			for id, deps := range IDDep {
				for d := range deps {
					if _, ok := IDDep[d]; !ok {
						remove = append(remove, pair{id, d})
					}
				}
			}
			for _, item := range remove {
				delete(IDDep[item.ID], item.Dep)
			}

			for id, deps := range IDDep {
				if 0 == len(deps) {
					loopID = append(loopID, id)
				}
			}
			if 0 == len(loopID) {
				err := ErrCircular{IDDep}
				return err
			}
		}

		// Move items from the lookup to the output layer.
		layer := make([]string, 0, len(loopID))
		for _, id := range loopID {
			delete(IDDep, id)
			layer = append(layer, id)

			if deps, ok := DepID[id]; ok {
				for dep, _ := range deps {
					delete(IDDep[dep], id)
				}
			}
		}
		layers = append(layers, layer)
	}

	// Ensure a consistent order.

	posMap := make(map[string]int, nc.Len())
	pos := 0
	for _, l := range layers {
		sort.Slice(l, func(i, j int) bool {
			return l[i] < l[j]
		})
		for _, id := range l {
			posMap[id] = pos
			pos++
		}
	}

	sort.Sort(sortableNodeCollection{
		NodeCollection: nc,
		pos:            posMap,
	})

	return nil
}
