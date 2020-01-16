package parse

import "io"

// TODO(daniel.theophanes): This is a place holder code to try to use the
// parse2 data types to produce reasonable SQL structure.

type CRDB struct{}

func (CRDB) GenerateSchema(root *Root, w io.Writer) error {
	// 1. Read database schema.
	// 2. Load database dependencies and sort dependencies (topological sort, alpha sort inside layers).
	// 3. Print each table in SQL dialect.

	// When resolving symbols and values, I need to take a reference: "foo.bar.blah"
	// and turn it into something useful like "Ref = *Blah" (I think).
	// I need to validate numbers and strings, and ensure each value is by itself in a cell.
	// If there is a table type, I need to resolve the table type to a table schema that can be checked.
	// Either: identifier | number | string | table
	// identifier: part[.part...]
	// number: 123.456
	// string: "ABC"
	// table: identifier(...)
	// query: ...
	//
	// Maybe add a new statement type "var". Store in separate namespace. Yes.
	// Assigning a variable to data is done by value, not by reference.
	//
	// Maybe allow appending to a type list like "schema table.types | jsonb" which would append jsonb to the existing schema name "types".
	//
	// First we lookup the schema to understand what type is expected.
	// Then if the type expects some value (number | string | table | query | ...), then we set the var.
	// If the type expects some reference (like to another table or table column) then we link to the data.
	//
	// 1. Order of declaration should not matter within a "thing" (module or package or something).
	// 2. Packages or modules or something need to have explicit imports and dependencies and be unble to modify everything.
	//
	// A schema should define what possible schema names.

	for _, db := range root.Data.Names["database"].Order {
		_ = db.Name

		for _, table := range db.Names["table"].Order {
			_ = table

			cols := table.Names["columns"].Value
			for _, c := range cols.Table.Order {
				// Most of this seems boring enough.
				// What if we unmarshalled into structs?
				// We can use a struct tag to look up the value,
				// Also, this can use the Schema lookup for things such as defaults and types,
				// though the Go types could also be used for types. Or both could be used to ensure
				// compatibility with other tools too.
				//
				// This may also work to automatically marshal the query format into structs as well.
				//
				// So perhaps the process to introduce a new type / UI would be to bind a new Go type to it?
				// Plus some schema annotations? That doesn't sound too difficult.

				// c.Names
				_ = c.Order[0].Order[0].Text
			}
		}
	}

	return nil
}

type QueryParam struct {
	Name  string
	Value interface{}
}

type QueryInput struct {
	Portal string
	SCD    string // SCD favor SQL.
	Params []QueryParam
}

type QueryOutput struct {
	SQL    string
	Params []QueryParam
}

func (CRDB) GenerateQuery(root *Root, input QueryInput) (QueryOutput, error) {
	qo := QueryOutput{}

	// 1. Lookup portal.
	// 2. Parse SCD.
	// 3. Resolve symbols, tables, and types.
	// 4. Load into a syntax tree.
	// 5. Print syntax tree in SQL dialect.

	return qo, nil
}
