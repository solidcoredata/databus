package library

Table :: {
	name: string
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

Field :: {
	name: string
	altname: [...string]
	order:    int
	type:     "int" | "text" | "decimal" | "float" | "bool" | "bytes" | "date" | "datetime" | "datetimez"
	fk:       string | *""
	nullable: bool | *false
	key:      bool | *false
	comment:  string | *""
	length:   int | *0
}

tables: [TableName=string] :: Table & {
	name: TableName
}

audit :: Table & {
	tags: audit: true
	columns: {
		timecreated: {type: "datetimez", order: 9000}
		timeupdated: {type: "datetimez", order: 9001}
	}
}
softdelete :: Table & {
	tags: softdelete: true
	columns: {
		deleted: {order: 8000, type: "bool"}
	}
}

tables: books: audit & softdelete & {
	tags: secret: true
	columns: {
		id: {order: 1, type: "int", key: true}
		bookname: {order: 2, type: "text", length: 400}
		pages: {order: 3, type: "int"}
		genre: {order: 4, type: "int", fk: "genre.id"}
	}
}

tables: genre: softdelete & {
	columns: {
		id: {order: 1, type: "int", key: true}
		name: {order: 2, type: "text", length: 400}
	}
}
