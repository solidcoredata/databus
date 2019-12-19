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
	R: 4
	I: 2
	D: 1
}

PortalTable :: {
	name: string
	require: [Role=uint]: [Permission=uint & <=7]: true | {
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
			"\(P.D)": {
				vars: ["account"]
				query: "b.published == false"
			}
		}
	}
}

// NOTE(daniel.theophanes): I think it is highly likely that these query definitions won't work.
Query :: {
	name:  string
	parts: _
}

queries: joina: Query & {
	name: "joina"
	parts: {
		a: {
			from: {
				b: {table: "books"}
				g: {table: "genre", on: "g.id = b.genre"}
			}
		}
	}
}
