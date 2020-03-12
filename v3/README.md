# Databus v3

3rd attempt of the most recent attempt at the databus.

## Design

There are two types of values:
 - structs denoted with `{}`
 - lists denoted with `()`

Struct or list types may be implied for nested values, or
stated prior to the bracket. For example `db.table{...}` is
a struct of type `db.table`. Where as `db.column(...)` is
a list of type `db.column`. If the type is known from a parent type,
the value may omit the type name and simply write `(...)` or `{...}`.

Every value is assigned to both a identifier path and an index (like a sorted map). Furthermore
each value has a type.

### Lists

There are two types of lists: simple lists and tables.
 - A simple list is a list of values that are separated by "," or end-of-statement (end-of-line or ";").
 - A table is an ordered key-value pairs that contain a "|" in the first "value".

A simple list example: `list(a, b, c)`.
A table example:
```
table(
     | attr1  | attr2
key1 | valueA | valueB
key2 | valueC | valueD
)
```
Results in the following value list:
```
[0].key1.attr1 = valueA
[0].key1.attr2 = valueB
[1].key2.attr1 = valueC
[1].key2.attr2 = valueD
```

### Structs

Structs always have a key to value list pair. Like:
 - `<key-X> <value>`
 - `<key-Y> <value-1> <value-2>`
 - `<key-Z> <value-A-1> <value-A-2>, <value-B-1> <value-B-2>`

These three lines would be translated into the following sequence:
```
$[0] = key-X
$[0]key-X[0] = value
$[1] = key-Y
$[1]key-Y[0] = value-1
$[1]key-Y[1] = value-2
$[2] = key-Z
$[2]key-Z[0] = value-A-1
$[2]key-Z[1] = value-A-2
$[3] = key-Z
$[3]key-Z[0] = value-B-1
$[3]key-Z[1] = value-B-2
```

In this example `key-Z` is evaluated to two indexes. In a struct, a comma will
sepearate value lines but repeat the key identifier.

### File

The top file level is defined to be a struct type in the parser, thus
the first identifier of every line is the file struct identifier.

### Schema

Data structures can be parsed lexically without knowledge of the types.
The file defines a basic file type that allows importing additional types.

Every value has a schema associated with it. The order in which values and
schemas generally shouldn't matter. A streaming parser may require all schemas
declared first, but most file based parsers shouldn't depend on declaration order
of types or schemas.


```
type schema {
	Value schema.value
	Accept and(schema)
}
type schema.value {
	Key identifier
	Arity ZeroMore
	Accept and(schema)
}

// Pre-declared schema types:
type text schema {
	Values (
		Cap ZeroOne uint // Max runes to fit.
	)
	Accept or(text)
}
type bool schema {
	Accept or(bool)
}
type dec schema {
	Values (
		Precision ZeroOne uint
		Scale ZeroOne uint
		Min ZeroOne dec
		Max ZeroOne dec
		RoundingMode ZeroOne or(ToNearestEven, ToNearestAway, ToZero, AwayFromZero, ToNegativeInf, ToPositiveInf, ToNearestTowardZero)
	)
	Accept or(dec, int, float)
}
type int schema {
	Values (
		Min ZeroOne int
		Max ZeroOne int
	)
	Accept or(int)
}
type float32
type float64

// Pre-declared derived types.
type uint int{Min 0}
type byte int {Min 0; Max 255}
type uint8  int {Min 0; Max 255}
type uint16
type uint32
type uint64
type int8
type int16
type int32
type int64
```

### Imports

You may import other packages that extend other packages. However, this doesn't
globally extend the types, it just extends them for that package. This can be
done because schemas are a validation, not storage, step.

### Queries

Struct
StructKey
Value
List

```
set SimpleSelect query{
	from books b
	from genre g
	and eq (b.published, true)
	select b.id, b.bookname
	select g.name genrename
	and eq(b.deleted, false)
}
```
file(struct)[0]
	set(key)[0]
		SimpleSelect(Value)[0]
		query(value)[1]
		(struct)[2]
			from(key)[0]
				books(Value)[0]
				b(Value)[1]
			from(key)[1]
				genre(Value)[0]
				g(Value)[1]
			and(key)[2]
				eq(Value)[0]
				(List)[1]
					b.published(Value)[0]
					true(Value)[1]
			select(key)[3]
				b.id(Value)[0]
			select(key)[4]
				b.bookname(Value)[1]
			select(key)[5]
				g.name(Value)[0]
				genrename(Value)[1]
			and(key)[6]
				eq(Value)[0]
				(List)[1]
					b.deleted(Value)[0]
					false(Value)[1]
=>

[0] = set
[0]set[0] = SimpleSelect
[0]set[1] = struct(query)
[0]set[1]struct(query)[0] = from
[0]set[1]struct(query)[0]from[0] = books
[0]set[1]struct(query)[0]from[1] = b
[0]set[1]struct(query)[1] = from
[0]set[1]struct(query)[1]from[0] = genre
[0]set[1]struct(query)[1]from[1] = g

[0] = set
[0]set[0] = SimpleSelect(Type=query,Group=struct)


```
SimpleSelect = query
SimpleSelect[0] = from
SimpleSelect[0].from[0] = books
SimpleSelect[0].from[1] = b
SimpleSelect[1] = from
SimpleSelect[1].from[0] = genre
SimpleSelect[1].from[1] = g
SimpleSelect[2] = and
SimpleSelect[2].and = eq
SimpleSelect[2].and[0].eq[0] = b.published
SimpleSelect[2].and[0].eq[1] = true
SimpleSelect[3] = select
SimpleSelect[3].select[0] = b.id
SimpleSelect[4] = select
SimpleSelect[4].select[0] = b.bookname
SimpleSelect[5] = select
SimpleSelect[5].select[0] = g.name
SimpleSelect[5].select[1] = genrename
SimpleSelect[6] = and
SimpleSelect[6].and = eq
SimpleSelect[6].and[0].eq[0] = b.deleted
SimpleSelect[6].and[0].eq[1] = false
```

## Exmple

Options:
 1. Put name and relationship inside definition. This resolves many questions.
 2. Continue to try to use `set identifier db.table{}` method. This has many questions.

Doing (1) simplifies many questions. This would probably be the fastest way forward.
It would also make it simple to have two notations for exported data and internal data.
However, both can be used with the current lex and parse method.

Then you would:
 1. Import a package that adds a schema to the file root.
 2. Decalre types (tables, columns, screens) to the file.
 3. When exporting the data, the public types are then processed. The export can be filtered by type.



## Runtime

### Internal

Import all values into a database table similar to:
```
create table kv (
	id integer
	parent> kv.id
	sort_order integer
	name text
	type_name
	type_link> kv.id nullable
	value text
)
```

### Verify

The format should verify itself easily.

### Import

 * Read existing running database and import the schema and procedures.
   - SQL Server

### Export

 * scd fmt
 * JSON
 * SQL (schema and queries)
   - SQL Server
   - CRDB
   - MySQL
   - PostgreSQL

### Binary Stream

Library to take an arbitrary SCD expression and efficently stream in over the wire.

## MVP

 * scd fmt
 * SQL Schema Export for CRDB or sqlite (easy to test and verify)
 * JSON Export
 * Simple identifier verifications

### Steps

 1. Write context free lexer parser that outputs a normalized list of `identifier = value`.
 2. Insert each of these rows into a database.
 3. Try to write a SQL Schema output.
 4. Try to write a JSON output.
 5. Try to verify type schemas.
 6. Try to write a scd fmt.

