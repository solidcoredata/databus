package library

import (
	"strings"
)

Table :: {
	name:  string
	alias: string | *strings.Split(name, "")[0]
	altname: [...string]
	tags: [TagName=string]:      _
	columns: [FieldName=string]: Field & {
		name: FieldName
	}
	indices: [IndexName=string]: Index & {
		name: IndexName
	}
}

Index :: {
	name: string
	index: [...string]
	include: [...string]
}

Type :: {
	int:       "int"
	text:      "text"
	decimal:   "decimal"
	float:     "float"
	bool:      "bool"
	bytes:     "bytes"
	date:      "date"
	datetime:  "datetime"
	datetimez: "datetimez"
}
T = Type

FieldType :: T.int | T.text | T.decimal | T.float | T.bool | T.bytes | T.date | T.datetime | T.datetimez

Field :: {
	name: string
	altname: [...string]
	order:    int
	type:     FieldType
	fk:       string | *""
	nullable: bool | *false
	key:      bool | *false
	comment:  string | *""
	length:   int | *0
}

Database :: {
	name: string
}

database: Database & {
	name: "library"
}

tables: [TableName=string] :: Table & {
	name: TableName
}

audit :: Table & {
	tags: audit: true
	columns: {
		timecreated: {type: T.datetimez, order: 9000}
		timeupdated: {type: T.datetimez, order: 9001}
	}
}
softdelete :: Table & {
	tags: softdelete: true
	columns: {
        // TODO(daniel.theophanes): add in a query clause that addes "deleted == false" unless ".deleted == true" and and role has RH.
		deleted: {order: 8000, type: T.bool}
	}
}

tables: books: audit & softdelete & {
	tags: secret: true
	columns: {
		id: {order: 1, type: T.int, key: true}
		bookname: {order: 2, type: T.text, length: 400}
		pages: {order: 3, type: T.int}
		genre: {order: 4, type: T.int, fk: "genre.id"}
		published: {order: 5, type: T.bool}
	}
}

tables: genre: softdelete & {
	columns: {
		id: {order: 1, type: T.int, key: true}
		name: {order: 2, type: T.text, length: 400}
	}
}

// TODO(daniel.theophanes): Define pre- and post- trigger actions on tables.
// TODO(daniel.theophanes): Define named portals, each with a set of hierarchical roles and per table accessors.
// TODO(daniel.theophanes): Define UI that can connect to tables.
// TODO(daniel.theophanes): Define query syntax for select and query shaping.

Portal :: {
	name: string
	roles: [RoleName=string]:   uint
	tables: [TableName=string]: PortalTable
}

P :: {
	RH: 16 // ReadHistory (Allows viewing soft deleted and history values.)
	R: 8  // Read
	I: 4  // Insert
    U: 2  // Update
	D: 1  // Delete (soft and hard)
}

PortalTable :: {
	name: string
	require: [Role=uint]: [Permission=uint & < 32]: true | {
		vars: [...string]
		query: string
	}
	// TODO(daniel.theophanes): Add in mandatory queries for a given role that will check permission to access.
}

portals: [PortalName=string]: {
	name: PortalName
}
portals: public: {
	roles: anonymous: 1
	tables: books: {
		require: "\(roles.anonymous)": {
			"\(P.R)": {
				vars: []
				query: "b.published == true"
			}
		}
	}
}
portals: admin: {
	roles: superadmin: 99
	roles: admin:      10
	roles: ro:         1
	tables: books: {
		require: "\(roles.superadmin)": {
			"\(P.R)": true
			"\(P.I)": true
			"\(P.U)": true
			"\(P.D)": {
				vars: ["account"]
				query: "b.published == false"
			}
		}
	}
}

// NOTE(daniel.theophanes): These queries are mostly for thinking about.
Query :: {
	name:  string
	parts: _
}

queries: joina: Query & {
	name: "joina"
	parts: {
        // TODO(daniel.theophanes): determine different join types (arity, inner join, cross join, left join) syntax.
        // TODO(daniel.theophanes): determine how to specify joins columns manually.
		SimpleSelect: """
            from books b
            from genre g
            and b.published == true
            select b.id, b.bookname
            select genrename = g.name
            // "and b.deleted == false" is automatically added unless otherwise specified.
        """
		SimpleInsert: """
            from books b
            and b.id == 12 // Duplicate this row
            // "and b.deleted == false" is automatically added unless otherwise specified.
            insert b
            set bookname = b.bookname
            set pages = b.pages
            set genre = .genre // This is an input parameter.
            set published = false
        """
		SimpleUpdate: """
            from books b
            and b.id == 12 // Update this row
            // "and b.deleted == false" is automatically added unless otherwise specified.
            update b // or just "update", which would refer to "from books b".
            set pages = .pages // Set pages to a new value from an input parameter.
        """
		SimpleDelete: """
            from books b
            and b.id == 12 // Delete this row
            // "and b.deleted == false" is automatically added unless otherwise specified.
            delete b
        """

        // General pattern for queries will be to name query parts, then reference
        // other query query parts from the from clause like any other table.
        // This make essetnally is a combination of table variables and CTEs,
        // implementations may use neither, either, or both.
        //
        // Need a simple syntax to assign columns from one query part to
        // the input parameters of another referenced query part.

		SelectGenre: """
            from genre g
            and g.id = .genre
            select genrename = g.name
        """
		SelectBook: """
            from books b
            from SelectGenre sg
            and b.published == true
            and b.genre == sg.genre
            select b.id, b.bookname
            select sg.genrename
        """
        // or
		SelectGenre: """
            from genre g
            and g.id = .genre
            select genrename = g.name
        """
		SelectBook: """
            from books b
            and b.published == true
            and exists ( // Restrict to a sub-query.
                from SelectGenre sg
                and sg.genre = b.genre
                and sg.genrename = 'History'
            )
            select b.id, b.bookname
        """
        // This also has the effect that each table is automatically named,
        // which is also desirable. Lastly, perhaps a ":" vs "::" distinction
        // could be made so that "::" is a "view" and ":" is an output.


		FullExample: """
            Query1:
            from books b
            and b.published == true
            select b.id, b.bookname

            ViewA::
            from books b
            and b.published == .published
            select b.genre

            Query2:
            from genre g
            select g.*
            and exists (
                from ViewA v
                and v.genre = g.id
                and v.published = 1
            )
        """
	}
}
