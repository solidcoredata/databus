package main

import (
	"fmt"
	"log"
	"net/http"
)

func main() {
	fmt.Println("simulate data bus")
	uinode := setupUINode()
	datanode := setupDataNode()
	bind := &Bind{Name: "books", Together: []*BusNode{uinode, datanode}}
	a := &app{
		Bind: []*Bind{bind},
	}
	err := a.Validate()
	if err != nil {
		log.Print(err)
	}

	http.ListenAndServe(":8080", a)
}

type app struct {
	Bind []*Bind
}

func (a *app) Validate() error {
	for _, b := range a.Bind {
		err := b.Validate()
		if err != nil {
			return err
		}

	}
	return nil
}
func (a *app) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	for _, b := range a.Bind {
		for _, bn := range b.Together {
			switch bn.Type {
			default:
				// Nothing for now.
			case "solidcoredata.org/data/table":
			case "solidcoredata.org/ui/sld":
			}

		}
	}

	fmt.Fprintf(w, "I'm alive!\nerrs: %v\n", a.Validate())
}

type Bind struct {
	Name     string
	Together []*BusNode

	errors []error
}

func (b *Bind) errorf(f string, v ...interface{}) {
	e := fmt.Errorf(f, v...)
	b.errors = append(b.errors, e)
}
func (b *Bind) Err() error {
	if len(b.errors) == 0 {
		return nil
	}
	return fmt.Errorf("%q", b.errors)
}

func (b *Bind) Validate() error {
	for _, bn := range b.Together {
		bn.roleName = make(map[string]*Role, len(bn.Roles))
		for ri := range bn.Roles {
			r := &bn.Roles[ri]
			bn.roleName[r.Name] = r
			if len(r.Bind) == 0 {
				continue
			}
			r.fieldBind = make(map[interface{}]bool, len(r.Fields))
			for _, f := range r.Fields {
				fv, ok := f[r.Bind]
				if !ok {
					continue
				}
				r.fieldBind[fv] = true
			}
		}
	}
	for bni, bn := range b.Together {
		if bni+1 >= len(b.Together) {
			continue
		}
		next := b.Together[bni+1]
		for _, r := range bn.Roles {
			if len(r.Bind) == 0 {
				continue
			}
			nextRole, ok := next.roleName[r.Name]
			if !ok {
				b.errorf("Unable to find role %q when binding types %s to %s", r.Name, bn.Type, next.Type)
				continue
			}
			for fb := range r.fieldBind {
				if !nextRole.fieldBind[fb] {
					b.errorf("Node Type %s Role %q: missing bound field %v", bn.Type, r.Name, fb)
					continue
				}
			}
		}

	}
	return b.Err()
}

type Property struct {
	Name     string
	Type     string
	Optional bool
}

type KV = map[string]interface{}

type Role struct {
	Name       string
	Properties []Property
	Fields     []KV
	Bind       string

	fieldBind map[interface{}]bool
}

type BusNode struct {
	Type  string
	Roles []Role

	roleName map[string]*Role
}

func setupUINode() *BusNode {
	return &BusNode{
		Type: "solidcoredata.org/ui/sld",
		Roles: []Role{
			{
				Name: "props1",
				Properties: []Property{
					{Name: "hide_new", Type: "bool"},
					{Name: "hide_edit", Type: "bool"},
					{Name: "hide_delete", Type: "bool"},
				},
				Fields: []KV{
					{"hide_new": false},
					{"hide_edit": false},
					{"hide_delete": false},
				},
			},
			{
				Name: "params",
				Properties: []Property{
					{Name: "name", Type: "text"},
				},
				Fields: []KV{
					{"name": "title_contains"},
				},
				Bind: "name",
			},
			{
				Name: "arity",
				Properties: []Property{
					{Name: "name", Type: "text"},
					{Name: "display", Type: "text"},
					{Name: "width", Type: "int"},
				},
				Fields: []KV{
					{"name": "title", "Display": "Book Title", "Width": 200},
					{"name": "author", "Display": "Book Author", "Width": 100},
				},
				Bind: "name",
			},
		},
	}
}

func setupDataNode() *BusNode {
	return &BusNode{
		Type: "solidcoredata.org/data/table",
		Roles: []Role{
			{
				Name: "data",
				Properties: []Property{
					{Name: "table", Type: "text"},
				},
				Fields: []KV{
					{"table": "books"},
				},
			},
			{
				Name: "params",
				Properties: []Property{
					{Name: "name", Type: "text"},
				},
				Fields: []KV{
					{"name": "title_contains"},
				},
				Bind: "name",
			},
			{
				Name: "arity",
				Properties: []Property{
					{Name: "name", Type: "text"},
				},
				Fields: []KV{
					{"name": "title"},
					{"name": "author"},
				},
				Bind: "name",
			},
		},
	}
}
