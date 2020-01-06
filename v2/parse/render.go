package parse

import "io"

// TODO(daniel.theophanes): This is a place holder code to try to use the
// parse2 data types to produce reasonable SQL structure.

type CRDB struct{}

func (CRDB) GenerateSchema(root *Root, w io.Writer) error {
	// 1. Read database schema.
	// 2. Load database dependencies and sort dependencies (topological sort, alpha sort inside layers).
	// 3. Print each table in SQL dialect.

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
