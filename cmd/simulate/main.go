package main

import (
	"fmt"
	"net/http"
)

func main() {
	fmt.Println("simulate data bus")
	uinode := setupUINode()
	datanode := setupDataNode()
	bind := &Bind{Together: []*BusNode{uinode, datanode}}
	a := &app{
		Bind: []*Bind{bind},
	}
	http.ListenAndServe(":8080", a)
}

type app struct {
	Bind []*Bind
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

	w.Write([]byte("I'm alive!"))
}

type Bind struct {
	Together []*BusNode
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
}

type BusNode struct {
	Type  string
	Roles []Role
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
			},
		},
	}
}
