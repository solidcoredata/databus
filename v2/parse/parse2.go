package parse

import (
	"context"
	"database/sql"
	"encoding/binary"
	"fmt"

	"github.com/golang-sql/table"
	"golang.org/x/crypto/blake2b"
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

var keyhashkey = []byte("solidcoredata")

func stringID(v string) uint64 {
	keyhash, err := blake2b.New(6, keyhashkey)
	if err != nil {
		panic(err)
	}
	keyhash.Write([]byte(v))
	sum := keyhash.Sum(nil)
	sum2 := make([]byte, 8)
	copy(sum2, sum)
	return binary.LittleEndian.Uint64(sum2)
}

func Parse3(pr *parseRoot) (*Root, error) {
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		return nil, err
	}
	defer db.Close()

	ctx := context.Background()

	t, err := table.NewBuffer(ctx, db, `
begin transaction;

create table filename (
	name text
);
create table pos (
	filename references filename(rowid),
	line_start integer,
	line_rune_start integer,
	line_end integer,
	line_rune_end ineger
);
create table x_db (
	rowid bigint priamry key,
	name text,
	pos references pos(rowid)
);
create table x_type (
	name text,
	pos references pos(rowid)
);
create table x_type_property (
	x_type references x_type(rowid),
	key text,
	value blob,
	pos references pos(rowid)
);
create table x_table (
	xdb integer references x_db(rowid),
	name text,
	pos references pos(rowid)
);
create table x_table_tag (
	xdb integer references x_table(rowid),
	name text,
	pos references pos(rowid)
);
create table x_table_column (
	xtable integer references x_table(rowid),
	name text,
	xtype integer references x_type(rowid),
	pos references pos(rowid)
);
insert into x_db (rowid, name)
values (66192772841080, 'library');

select * from x_db;

commit transaction;
	`)
	if err != nil {
		return nil, err
	}

	dbid := stringID("library")
	fmt.Println("library", dbid)
	dbidx := stringID("librarylibrary")
	fmt.Println("librarylibrary", dbidx)

	for _, row := range t.Rows {
		fmt.Println("row: ", row)
	}

	return nil, nil
}

func Parse4(pr *parseRoot) (*Root, error) {
	type Valuer interface {
		Value() interface{}
		Span() (start pos, end pos)
	}
	type P struct {
		Key   string
		Value Valuer
	}
	type Z struct {
		Value      Valuer
		Index      int32
		Properties []P // Probably remove.
	}
	type QueryLine struct {
		Valuer
		Verb   string // from, select, and.
		Values []Valuer
	}
	// Query must itself implement the valuer interface.
	type Query struct {
		Valuer
		Lines []QueryLine
	}
	type Root struct {
		FullPath map[string]*Z
	}
	return nil, nil
}
func Parse2(pr *parseRoot) (*Root, error) {
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
