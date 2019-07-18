package inter

import (
	"bytes"
	"context"
	"fmt"
	"strings"

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

func (cr *CRDB) AboutSelf() ExtensionAbout {
	return ExtensionAbout{
		Name: "crdb",
		HandleTypes: []string{
			typeSQLDatabase,
			typeSQLTable,
		},
	}
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
func (cr *CRDB) Generate(ctx context.Context, delta *bus.DeltaBus, writeFile ExtensionVersionWriter) error {
	b := delta.Current
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
				fName := f.Value("name").(string)
				fType := f.Value("type").(string)
				fNull := f.Value("nullable").(bool)
				fLength := f.Value("length").(int64)
				fFKRaw := f.Value("fk")
				fKey := f.Value("key").(bool)
				fComment := f.Value("comment").(string)
				fDisplay := f.Value("display").(string)
				/*
				   {Name: "name", Type: "text", Optional: false, Send: true, Recv: false}, // Database column name.
				   {Name: "display", Type: "text", Optional: false, Send: true, Recv: false}, // Display name to default to when displaying data from this field.
				   {Name: "type", Type: "text", Optional: false, Send: true, Recv: false}, // Type of the database field.
				   {Name: "fk", Type: "node", Optional: true, Send: false, Recv: false},
				   {Name: "length", Type: "int", Optional: true, Send: true, Recv: false}, // Max length in runes (text) or bytes (bytea).
				   {Name: "nullable", Type: "bool", Optional: true, Send: false, Recv: false, Default: "false"}, // True if the column should be nullable.
				   {Name: "key", Type: "bool", Optional: true, Send: false, Recv: false, Default: "false"}, // True if the column should be nullable.
				   {Name: "comment", Type: "text", Optional: true, Send: false, Recv: false},
				*/
				if len(fComment) > 0 {
					w("\n\t-- %s", strings.ReplaceAll(fComment, "\n", "\n\t-- "))
				}
				if len(fDisplay) > 0 {
					w("\n\t-- Display: %s", fDisplay)
				}
				var dbType string
				switch fType {
				default:
					dbType = fType
				case "text":
					dbType = "string"
				}
				w("\n\t%s %s", fName, dbType)
				if fLength > 0 {
					w("(%d)", fLength)
				}
				if fNull {
					w(" null")
				} else {
					w(" not null")
				}
				if fKey {
					w(" primary key")
				}
				if fFKRaw != nil {
					// TODO
				}
			}
			w("\n);\n")
		}
	}
	return writeFile(ctx, "schema.sql", buf.Bytes())
}

// Read generated files and deploy to system.
func (cr *CRDB) Deploy(ctx context.Context, opts *DeployOptions, delta *bus.DeltaBus, readFile ExtensionVersionReader) error {
	panic("TODO")
}
