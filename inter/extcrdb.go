package inter

import (
	"bytes"
	"context"
	"fmt"

	"solidcoredata.org/src/databus/bus"
	"solidcoredata.org/src/databus/internal/tsort"
)

const (
	typeSQLDatabase = "solidcoredata.org/t/db/database"
	typeSQLTable    = "solidcoredata.org/t/db/table"
)

func NewCRDB() *CRDB {
	return &CRDB{}
}

var _ Extension = &CRDB{}

type CRDB struct{}

func (cr *CRDB) AboutSelf(ctx context.Context) (ExtensionAbout, error) {
	return ExtensionAbout{
		Name: "crdb",
	}, nil
}

// Extension specific Bus validation.
func (cr *CRDB) Validate(ctx context.Context, b *bus.Bus) error {
	return nil
}

var _ tsort.NodeCollection = (*bussort)(&bus.Bus{})

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

// Generate and write files. Note, no file list is provided so extensions should
// write a manafest file of some type by a well known name.
func (cr *CRDB) Generate(ctx context.Context, b *bus.Bus, writeFile ExtensionVersionWriter) error {
	buf := &bytes.Buffer{}
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
	return writeFile(ctx, "schema.sql", buf.Bytes())
}

// Read generated files and deploy to system.
func (cr *CRDB) Deploy(ctx context.Context, opts *DeployOptions, b *bus.Bus, readFile ExtensionVersionReader) error {
	panic("TODO")
}
