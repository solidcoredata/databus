# Databus v3

3rd attempt of the most recent attempt at the databus.

## Design

To avoid nesting, each file is a series of statements:
```
<verb> <path> [<value>]
```

There are two types of values:
 - structs denoted with `{}`
 - lists denoted with `()`

Struct or list types may be implied for nested values, or
stated prior to the bracket. For example `db.table{...}` is
a struct of type `db.table`. Where as `db.column(...)` is
a list of type `db.column`. If the type is known from a parent type,
the value may omit the type name and simply write `(...)` or `{...}`.

Furthermore, there are two types of lists: simple lists and tables.
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
table[0].key1.attr1 = valueA
table[0].key1.attr2 = valueB
table[1].key2.attr1 = valueC
table[1].key2.attr2 = valueD
```

The list of top level verbs include:
 - package
 - context
 - set
 - (type or schema)?

## Exmple

