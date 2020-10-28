set db.table type {
	tag 0..N db.tag
	column 1..N db.column
}

set db.tag type {}

set db.column type {
	0:name 1 id
	1:type: 1 db.type
	2:default 0..1 or (
		value,
		query,
	)
}

set query.from type {
	0:name 1 or(
		db.table
		db.view
		db.query
		query
	)
	1:alias 1 id
}
set query.and type {
	0:exp or(
		value
		expression
		function
		query
	)
}
set query.select type {
	0:ref 1 or (
		value
		query
		db.table.column
	)
	1:alias 0..1 id
}

set table.book db.table {
	tag softdelete, audit
	column db.column {name published; type bool; default false;}
	column db.column {name "read only"; type bool; default false;}
}

set SimpleSelect query {
	from   books b
	from   genre g
	and    eq(b.published, true)
	select b.id, b.bookname
	select g.name genre_name
	and    eq(b.deleted, false)
}

/*
	Create a DB Schema.
	Manually insert into the database schema.
	Type resove the schema.
	Try to output from the populated database:
	 - JSON
	 - SQL
	 - SCD

	If all works out above, work on translating the parsed text (or fluent API) into database inserts.
*/