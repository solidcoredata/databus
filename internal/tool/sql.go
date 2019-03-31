package tool

import (
	"bytes"
	"context"
	"fmt"
	"sort"

	"solidcoredata.org/src/databus/bus"
)

var _ bus.RunStart = &SQLGenerate{}

type SQLGenerate struct{}

const (
	typeSQLDatabase = "solidcoredata.org/t/db/database"
	typeSQLTable    = "solidcoredata.org/t/db/table"
)

func (s *SQLGenerate) NodeTypes(ctx context.Context, header *bus.CallHeader, request *bus.CallNodeTypesRequest) (*bus.CallNodeTypesResponse, error) {
	return &bus.CallNodeTypesResponse{
		CallVersion: 1,
		NodeTypes: []string{
			typeSQLDatabase,
			typeSQLTable,
		},
	}, nil
}

func (s *SQLGenerate) Run(ctx context.Context, header *bus.CallHeader, request *bus.CallRunRequest) (*bus.CallRunResponse, error) {
	c := request.Current
	buf := &bytes.Buffer{}
	switch variant := header.Options["variant"]; variant {
	default:
		return nil, fmt.Errorf("unknown SQL variant type %q", variant)
	case "cockroach":
		err := s.cockroach(c, buf)
		if err != nil {
			return nil, err
		}
	}
	err := outputFile(ctx, request.Root, "schema.sql", buf.Bytes())
	if err != nil {
		return nil, err
	}
	return &bus.CallRunResponse{
		CallVersion: 1,
	}, nil
}

func (s *SQLGenerate) cockroach(b *bus.Bus, buf *bytes.Buffer) error {
	nodes := b.Nodes
	// TODO(daniel.theophanes): Don't just sort by name, sort by reverse dependency order, then by name.
	sort.Slice(nodes, func(i, j int) bool {
		a, b := nodes[i], nodes[j]
		if a.Type == b.Type {
			return a.Name < b.Name
		}
		return a.Type < b.Type
	})
	w := func(s string, v ...interface{}) {
		fmt.Fprintf(buf, s, v...)
	}
	for _, n := range nodes {
		switch n.Type {
		default:
			return fmt.Errorf("unknown type: %q", n.Type)
		case typeSQLDatabase:
			prop := n.Role("prop").Fields[0]
			name := prop.Value("name").(string)
			w("create database %[1]s;\nset database = %[1]s;\n\n", name)
		case typeSQLTable:
			// TODO(daniel.theophanes): Also add in nullable, length limits, primary keys, and family.
			prop := n.Role("prop").Fields[0]
			name := prop.Value("name")
			db := prop.Value("database").(*bus.Node)
			_ = db

			sch := n.Role("schema")
			w("create table %s (", name)
			for i, f := range sch.Fields {
				if i != 0 {
					w(",")
				}
				fname := f.Value("name").(string)
				ftype := f.Value("type").(string)
				w("\n\t%s %s", fname, ftype)
			}
			w("\n);\n")
		}
	}
	return nil
}
