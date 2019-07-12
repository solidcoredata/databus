package tool

import (
	"bytes"
	"context"
	"fmt"

	"solidcoredata.org/src/databus/bus"
	"solidcoredata.org/src/databus/internal/tsort"
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
	case "crdb":
		err := s.cockroach(c, buf)
		if err != nil {
			return nil, err
		}
	}
	return &bus.CallRunResponse{
		CallVersion: 1,
		Files: []bus.CallRunFile{
			{Path: "schema.sql", Content: buf.Bytes()},
		},
	}, nil
}

var _ tsort.NodeCollection = (*bussort)(&bus.Bus{})

// TODO(daniel.theophanes): finish bussort implementation.
type bussort bus.Bus

func (bs *bussort) Index(i int) tsort.Node {
	return tsort.Node(bs.Nodes[i])
}
func (bs *bussort) Len() int {
	return len(bs.Nodes)
}
func (bs *bussort) Swap(i, j int) {
	bs.Nodes[i], bs.Nodes[j] = bs.Nodes[j], bs.Nodes[i]
}

func (s *SQLGenerate) cockroach(b *bus.Bus, buf *bytes.Buffer) error {
	err := tsort.Sort((*bussort)(b))
	if err != nil {
		return err
	}
	w := func(s string, v ...interface{}) {
		fmt.Fprintf(buf, s, v...)
	}
	for _, n := range b.Nodes {
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
