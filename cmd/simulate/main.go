package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
)

func main() {
	fmt.Println("simulate data bus")
	uinode := setupUINode()
	datanode := setupDataNode()
	indexnode := setupIndexNode()
	a := &app{
		Bind: []*Bind{
			&Bind{Name: "books", Together: []*BusNode{uinode, datanode}},
			&Bind{Name: "home", Together: []*BusNode{indexnode}},
		},
	}
	a.Validate()

	http.ListenAndServe(":8080", a)
}

const (
	typeSLD = `"use strict"
return {
	Render: function(bind, root) {
		console.log("sld", bind, root);
		root.innerText = "My books!";
	}
}
//# sourceURL=sld.js
`

	typeIndex = `"use strict"
function el(tag, prop) {
	let e = document.createElement(tag);
	if(prop.Text) {
		e.innerText = prop.Text;
	}
	if(prop.Parent) {
		prop.Parent.appendChild(e);
	}

	return e;
}
var body = null;
window.addEventListener("hashchange", function(ev) {
	console.log(ev.newURL, ev.oldURL);
	let name = location.hash.substring(2);
	window.__run.api.getBind(name, function(ok, bind, type) {
		if(!ok) {
			return;
		}
		type.Render(bind, body);
	});
});
return {
	Render: function(bind, root) {
		console.log(bind, root);
		root.innerHTML = "";
		let pages = bind.Role.pages;
		let nav = el("div", {Parent: root});
		body = el("div", {Parent: root});
		for(var i = 0; i < pages.length; i++) {
			let p = pages[i];
			let a = el("a", {Parent: nav, Text: p.display});
			a.href = "#!" + p.page;
		}
	}
}
//# sourceURL=index.js
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
		together := b.Together[0]
		rr := make(map[string]interface{}, len(together.Roles))
		for _, r := range together.Roles {
			rr[r.Name] = r.Fields
		}
		type uibind struct{ Type string; Role map[string]interface{} }
		json.NewEncoder(w).Encode(uibind{
			Type: together.Type,
			Role: rr,
		})

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


func setupIndexNode() *BusNode {
	return &BusNode{
		Type: "solidcoredata.org/ui/index",
		Roles: []Role{
			{
				Name: "pages",
				Properties: []Property{
					{Name: "page", Type: "text"},
					{Name: "display", Type: "text"},
				},
				Fields: []KV{
					{"page": "home", "display": "Home"},
					{"page": "books", "display": "Books"},
				},
			},
		},
	}
}
