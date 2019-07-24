package caller

import (
	"bytes"
	"context"
	"fmt"
	"strings"

	"solidcoredata.org/src/databus/bus"
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

// Generate and write files. Note, no file list is provided so extensions should
// write a manafest file of some type by a well known name.
func (cr *CRDB) Generate(ctx context.Context, delta *bus.DeltaBus, writeFile ExtensionVersionWriter) error {
	b := delta.Current
	buf := &bytes.Buffer{}
	w := func(s string, v ...interface{}) {
		fmt.Fprintf(buf, s, v...)
	}
	encodeFieldAttr := func(f *bus.Field) {
		fName := f.Name()
		fType := f.Value("type").(string)
		fNull := f.Value("nullable").(bool)
		fLength := f.Value("length").(int64)
		fFKRaw := f.Value("fk")
		fKey := f.Value("key").(bool)

		var dbType string
		switch fType {
		default:
			dbType = fType
		case "text":
			dbType = "string"
		}
		w("%s %s", fName, dbType)
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
			fk := fFKRaw.(*bus.Node)
			w(" references %s", fk.Role("prop").Fields[0].Value("name"))
		}
	}
	encodeField := func(f *bus.Field) {
		fComment := f.Value("comment").(string)
		fDisplay := f.Value("display").(string)

		if len(fComment) > 0 {
			w("\n\t-- %s", strings.ReplaceAll(fComment, "\n", "\n\t-- "))
		}
		if len(fDisplay) > 0 {
			w("\n\t-- Display: %s", fDisplay)
		}
		w("\n\t")
		encodeFieldAttr(f)
	}
	createNode := func(n *bus.Node) error {
		switch n.Type {
		default:
			return fmt.Errorf("unknown type: %q", n.Type)
		case typeSQLDatabase:
			prop := n.Role("prop").Fields[0]
			name := prop.Name()
			w("create database %[1]s;\nset database = %[1]s;\n\n", name)
		case typeSQLTable:
			prop := n.Role("prop").Fields[0]
			name := prop.Name()
			db := prop.Value("database").(*bus.Node)
			_ = db

			sch := n.Role("schema")
			w("create table %s (", name)
			for i := range sch.Fields {
				if i != 0 {
					w(",")
				}
				encodeField(&sch.Fields[i])
			}
			w("\n);\n")
		}
		return nil

	}
	for i := range b.Nodes {
		err := createNode(&b.Nodes[i])
		if err != nil {
			return err
		}
	}
	err := writeFile(ctx, "schema.sql", buf.Bytes())
	if err != nil {
		return err
	}
	buf.Reset()

	// w("begin transaction;\n")

	for _, alter := range delta.Actions {
		switch alter.Alter {
		default:
			return fmt.Errorf("unknown delta alter: %v", alter.Alter)
		case bus.AlterNothing:
			// Nothing.
		case bus.AlterScript:
			w("\n%s\n", alter.Script)
		case bus.AlterNodeAdd:
			err := createNode(alter.NodeCurrent)
			if err != nil {
				return err
			}
		case bus.AlterNodeRemove:
			n := alter.NodePrevious
			switch n.Type {
			default:
				return fmt.Errorf("unknown type: %q", n.Type)
			case typeSQLDatabase:
				prop := n.Role("prop").Fields[0]
				name := prop.Name()
				w("drop database %[1]s;\n", name)
			case typeSQLTable:
				prop := n.Role("prop").Fields[0]
				name := prop.Name()

				w("drop table %s;\n", name)
			}
		case bus.AlterNodeRename:
			n := alter.NodePrevious
			nTo := alter.NodeCurrent
			switch n.Type {
			default:
				return fmt.Errorf("unknown type: %q", n.Type)
			case typeSQLDatabase:
				prop := n.Role("prop").Fields[0]
				name := prop.Name()
				propTo := nTo.Role("prop").Fields[0]
				nameTo := propTo.Name()
				w("alter database %s rename to %s;\n", name, nameTo)
			case typeSQLTable:
				prop := n.Role("prop").Fields[0]
				name := prop.Name()
				propTo := nTo.Role("prop").Fields[0]
				nameTo := propTo.Name()

				w("alter table %s rename to %s;\n", name, nameTo)
			}
		case bus.AlterFieldAdd:
			n := alter.NodeCurrent
			switch n.Type {
			default:
				return fmt.Errorf("unknown type: %q", n.Type)
			case typeSQLDatabase:
				// Nothing.
			case typeSQLTable:
				prop := n.Role("prop").Fields[0]
				name := prop.Name()

				w("alter table %s add column", name)
				encodeFieldAttr(alter.FieldCurrent)
				w(";\n")
			}
		case bus.AlterFieldRemove:
			n := alter.NodeCurrent
			switch n.Type {
			default:
				return fmt.Errorf("unknown type: %q", n.Type)
			case typeSQLDatabase:
				// Nothing.
			case typeSQLTable:
				prop := n.Role("prop").Fields[0]
				name := prop.Name()
				fName := alter.FieldPrevious.Name()

				w("alter table %s drop column %s;\n", name, fName)
			}
		case bus.AlterFieldRename:
			n := alter.NodeCurrent
			switch n.Type {
			default:
				return fmt.Errorf("unknown type: %q", n.Type)
			case typeSQLDatabase:
				w("alter database %s rename to %s;\n", alter.FieldPrevious.Name(), alter.FieldCurrent.Name())
			case typeSQLTable:
				prop := n.Role("prop").Fields[0]
				name := prop.Value("name")

				w("alter table %s rename %s to %s;\n", name, alter.FieldPrevious.Name(), alter.FieldCurrent.Name())
			}
		case bus.AlterFieldUpdate:
			n := alter.NodeCurrent
			switch n.Type {
			default:
				return fmt.Errorf("unknown type: %q", n.Type)
			case typeSQLDatabase:
				// Nothing.
			case typeSQLTable:
				prop := n.Role("prop").Fields[0]
				name := prop.Value("name")

				w("alter table %s alter column ", name)
				encodeFieldAttr(alter.FieldCurrent)
				w(";\n")
			}
		}
	}

	// w("commit transaction;\n")

	return writeFile(ctx, "alter.sql", buf.Bytes())
}

// Read generated files and deploy to system.
func (cr *CRDB) Deploy(ctx context.Context, opts *DeployOptions, delta *bus.DeltaBus, readFile ExtensionVersionReader) error {
	panic("TODO")
}
