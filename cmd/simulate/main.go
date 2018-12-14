package main

import (
	"fmt"
	"log"
	"net/http"
	"strings"
)

func main() {
	fmt.Println("simulate data bus")
	uinode := setupUINode()
	datanode := setupDataNode()
	a := &app{
		Bind: []*Bind{
				&Bind{Name: "books", Together: []*BusNode{uinode, datanode}},
				&Bind{Name: "home", Together: []*BusNode{
					{Type: "solidcoredata.org/ui/index"},
				}},
		},
	}
	a.Validate()
	
	a.Types = map[string]TypeHandler{
		"solidcoredata.org/ui/index": func(all []*Bind, bind *Bind, w http.ResponseWriter, r *http.Request) {
			for _, b := range all {
				fmt.Fprintf(w, `<a href="?at=%s">%s</a><br>`, b.Name, strings.Title(b.Name))
			}

		},
		"solidcoredata.org/ui/sld": func(all []*Bind, bind *Bind, w http.ResponseWriter, r *http.Request) {
			bn := bind.Together[0]
			fmt.Fprintf(w, "SLD: %q", bn.Type)
		},
	}


	http.ListenAndServe(":8080", a)
}

type TypeHandler func(all []*Bind, bind *Bind, w http.ResponseWriter, r *http.Request)

type app struct {
	Bind []*Bind
	Types map[string]TypeHandler
	
	bindName map[string]*Bind
}

func (a *app) Validate() {
	a.bindName = make(map[string]*Bind, len(a.Bind))
	for _, b := range a.Bind {
		a.bindName[b.Name] = b
		err := b.Validate()
		if err != nil {
			log.Print(err)
		}
	}
}
func (a *app) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// First get the name of the current binding.
	// If empty name, show index.
	// If unknown name, show 404.
	// If matched, lookup type and pass control to that.
	at := r.URL.Query().Get("at")
	if len(at) == 0 {
		at = "home"
}

	b, found := a.bindName[at]
	if !found {
		http.NotFound(w, r)
		return
	}
	t, found := a.Types[b.Together[0].Type]
	if !found {
		http.NotFound(w, r)
		return
	}
	t(a.Bind, b, w, r)
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
func (b *Bind) Valid() bool {
	return len(b.errors) == 0
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
