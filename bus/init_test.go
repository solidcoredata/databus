package bus_test

import (
	"context"
	"strings"
	"testing"

	"solidcoredata.org/src/databus/bus/load"

	"github.com/google/go-jsonnet"
)

func TestVerify(t *testing.T) {
	var input = `
{
    Types: [
        {
            Name: "solidcoredata.org/test",
            Roles: [
                {
                    Name: "p1",
                    Properties: [
                        {Name: "name", Type: "text"},
                        {Name: "price", Type: "decimal"},
                        {Name: "related", Type: "node", Optional: true},
                    ],
                },
            ],
        },
    ],
    Nodes: [
        {
            Name: "node1",
            Type: "solidcoredata.org/test",
            Roles: [
                {
                    Name: "p1",
                    Fields: [
                        {KV: {name: "books", price: "123.456"}},
                    ],
                },
            ],
            Binds: [],
        },
        {
            Name: "node2",
            Type: "solidcoredata.org/test",
            Roles: [
                {
                    Name: "p1",
                    Fields: [
                        {Alias: "n1", KV: {name: "books", price: "123.456", related: "node1"}},
                    ],
                },
            ],
            Binds: [
                {Alias: "n1", Name: "node1"},
            ],
        },
    ],
}
    `

	ctx := context.Background()
	bus, err := load.BusReader(ctx, strings.NewReader(throughJsonnet(t, input)))
	if err != nil {
		t.Fatal("load", err)
	}
	err = bus.Init()
	if err != nil {
		t.Fatal("validate", err)
	}
}

// throughJsonnet is used to allow trailing commas in input, and
// allow most keys to be un-quoted. It also gives really good error messages.
func throughJsonnet(t *testing.T, s string) string {
	vm := jsonnet.MakeVM()
	out, err := vm.EvaluateSnippet("input", s)
	if err != nil {
		t.Fatal("jsonnet", err)
	}
	return out
}
