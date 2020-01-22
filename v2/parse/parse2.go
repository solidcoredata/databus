package parse

import (
	"context"
	"database/sql"
	"fmt"

	_ "modernc.org/sqlite"
)

type X struct {
	Name    string
	Order   []*X
	Names   map[string]*X
	Created bool
	Value   *DataAtom
}

func (x *X) relative(id *parseFullIdentifier) *X {
	return nil
}

func (x *X) Find(ref string) *X {
	return nil
}

type Root struct {
	Schema *X
	Data   *X
	Var    *X
}

func Parse2(pr *parseRoot) (*Root, error) {
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		return nil, err
	}
	defer db.Close()

	ctx := context.Background()

	_, err = db.ExecContext(ctx, `
create table xdb (
	name text
);
create table xtable (
	xdb integer references xdb(rowid),
	name text
);
create table xcolumn (
	xtable integer references xtable(rowid),
	name text
);
	`)
	if err != nil {
		return nil, err
	}

	return nil, nil

	root := &Root{
		Schema: &X{},
		Data:   &X{},
		Var:    &X{},
	}
	currentData := root.Data
	currentSchema := root.Schema
	currentVar := root.Var
	for _, st := range pr.Statements {
		switch st.Type {
		default:
			return nil, fmt.Errorf("unknown statement type: %v", st.Type)
		case statementContext:
			currentData = root.Data.relative(st.Identifier)
			currentSchema = root.Schema.relative(st.Identifier)
			currentVar = root.Var.relative(st.Identifier)
		case statementCreate:
			v := currentData.relative(st.Identifier)
			if v.Created {
				return nil, fmt.Errorf("already created")
			}
			v.Created = true
		case statementComment:
			// Ignore.
		case statementSet:
			v := currentData.relative(st.Identifier)
			da, err := DataAtomFromParseValueList(st.Value)
			if err != nil {
				return nil, err
			}
			v.Value = da
		case statementSchema:
			v := currentSchema.relative(st.Identifier)
			da, err := DataAtomFromParseValueList(st.Value)
			if err != nil {
				return nil, err
			}
			v.Value = da
		case statementVar:
			v := currentVar.relative(st.Identifier)
			da, err := DataAtomFromParseValueList(st.Value)
			if err != nil {
				return nil, err
			}
			v.Value = da
		}
	}
	return root, nil
}

type DataTable struct {
	Prefix      []lexToken
	Order       []*DataRow
	Names       map[string]*DataRow
	ColumnOrder []*DataColumn
	ColumnNames map[string]*DataColumn
}

type DataColumn struct {
	Name string
	// Type ?
}

type DataRow struct {
	Order []*DataValue
	Names map[string]*DataValue // map[column name]*DataValue
}

type DataValue struct {
	Order []*DataAtom
}

type DataNumber string

type DataAtom struct {
	Ref    *DataAtom
	Text   string
	Number DataNumber
	Table  *DataTable
	List   *DataRow
}

func DataAtomFromParseValueList(p *parseValueList) (*DataAtom, error) {
	return nil, nil
}
