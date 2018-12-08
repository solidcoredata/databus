package main

import (
	"fmt"
	"net/http"
)

func main() {
	fmt.Println("simulate data bus")
	a := &app{}
	http.ListenAndServe(":8080", a)
}

type app struct{}

func (a *app) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	w.Write([]byte("I'm alive!"))
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
	Roles []Role
}

func setupUINode() *BusNode {
	return &BusNode{
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
				},
				Fields: []KV{
					{"name": "title"},
					{"name": "author"},
				},
			},
		},
	}
}
