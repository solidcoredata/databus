// Package tsort implements a layered topological sort.
package tsort

import (
	"fmt"
	"sort"
	"strings"
)

// Node that may contain zero or more dependencies
type Node struct {
	ID           string
	Dependencies []string
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
	buf := &strings.Builder{}
	buf.WriteString("circular reference:\n")
	for key, set := range nset {
		buf.WriteString("- ")
		buf.WriteString(key)
		buf.WriteRune('\n')
		for dep := range set {
			buf.WriteString("--- ")
			buf.WriteString(dep)
			buf.WriteRune('\n')
		}
	}
	return buf.String()
}

// Sort the given nodes into layers.
func Sort(nodes []Node) (layers [][]string, err error) {
	IDDep := make(namedSet)
	DepID := make(namedSet)
	for _, n := range nodes {
		if _, ok := IDDep[n.ID]; ok {
			return nil, fmt.Errorf("Node %q is present more then once", n.ID)
		}
		IDDep[n.ID] = make(map[string]bool)
		for _, dep := range n.Dependencies {
			if dep == n.ID {
				continue
			}
			IDDep[n.ID][dep] = true
			if _, ok := DepID[dep]; !ok {
				DepID[dep] = make(map[string]bool)
			}
			DepID[dep][n.ID] = true
		}
	}

	type pair struct {
		ID, Dep string
	}

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
				return layers, err
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
	
	// Ensure consistent order is always returned.
	for _, l := range layers {
		sort.Slice(l, func(i, j int) bool {
			return l[i] < l[j]
		})
	}
	return layers, nil
}
