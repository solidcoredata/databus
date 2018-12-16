package main

import (
	"encoding/json"
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

const (
	typeSLD = `"use strict"
return {
	Render: function(bind, root) {
		console.log("sld", bind, root);
	}
}

`

	typeIndex = `"use strict"
return {
	Render: function(bind, root) {
		console.log("index", bind, root);
	}
}

`
	root = `<!DOCTYPE html>
Loading, please wait.
<script>
window.__run = (function(window) {
var run = {
	api:  {},
	type: {},
	bind: {},
};
function setStatus(msg) {
	if(msg.length === 0) {
		return;
	}
	if(console && console.log) {
		console.log(msg)
	}
}

run.api.getType = function(name, sub, done) {
	var req = new XMLHttpRequest();
	req.open("GET", "/type?name=" + encodeURIComponent(name) + "&sub=" + encodeURIComponent(sub), true);
	req.onreadystatechange = function () {
		if(req.readyState !== 4) {
			return;
		}
		if(req.status !== 200) {
			setStatus("Unable to contact server. Application may be down for maintenance: " + req.status);
			done(false);
			return;
		}
		setStatus("");

		var t = (new Function(req.responseText))();
		run.type[name] = t
		done(true, t);
	};
	req.send();
}
run.api.getBind = function(name, done) {
	var req = new XMLHttpRequest();
	req.open("POST", "/bind?name=" + encodeURIComponent(name), true);
	req.onreadystatechange = function () {
		if(req.readyState !== 4) {
			return;
		}
		if(req.status !== 200) {
			setStatus("Unable to contact server. Application may be down for maintenance: " + req.status);
			return;
		}
		setStatus("");

		var resp = JSON.parse(req.responseText);
		var b = resp;
		run.bind[name] = b;
		var t = run.type[resp.Type];
		if(t) {
			done(true, b, t);
			return;
		}
		run.api.getType(resp.Type, "", function(ok, t) {
			if(!ok) {
				done(false);
			}
			done(true, b, t);
		});
	};
	req.send();
}
return run;
})();
window.__run.api.getBind("home", function(ok, bind, type) {
	if(!ok) {
		return;
	}
	type.Render(bind, document.body);
});
</script>
`
)

type app struct {
	Bind  []*Bind
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
	switch r.URL.Path {
	default:
		http.NotFound(w, r)
		return
	case "/type":
		if r.Method != "GET" {
			http.NotFound(w, r)
			return
		}
		q := r.URL.Query()
		name := q.Get("name")
		sub := q.Get("sub")
		// TODO(daniel.theophanes): Transfer assets for control types.
		_, _ = name, sub
		switch name {
		default:
			http.NotFound(w, r)
			return
		case "solidcoredata.org/ui/sld":
			w.Write([]byte(typeSLD))
		case "solidcoredata.org/ui/index":
			w.Write([]byte(typeIndex))
		}

	case "/bind":
		if r.Method != "POST" {
			http.NotFound(w, r)
			return
		}
		q := r.URL.Query()
		name := q.Get("name")
		b, ok := a.bindName[name]
		if !ok {
			http.NotFound(w, r)
			return
		}
		// TODO(daniel.theophanes): Transfer configuration for bindings. Transfer all, or just computer (probably just collapsed computed).
		_ = b
		json.NewEncoder(w).Encode(struct{ Type string }{Type: b.Together[0].Type})

	case "/":
		if r.Method != "GET" {
			http.NotFound(w, r)
			return
		}
		// Transfer root index HTML and root javascript.
		w.Write([]byte(root))

	case "/bus":
		if r.Method != "POST" {
			http.NotFound(w, r)
			return
		}
		// TODO(daniel.theophanes): Transfer data to and from the client.

	}
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
				Fields: []KV{{
					"hide_new":    false,
					"hide_edit":   false,
					"hide_delete": false,
				}},
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
