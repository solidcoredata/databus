package main

import (
	"context"
)

/*
	// Root context

	// "CarMakes" is the table name.
	// "tags" is the label of this stanza, so a table can easily be declared in multiple sections.
	// ":db.table" is the type. This is most useful for top level values.
	let CarMakes tags :db.table {
		tag {|
			name
			HardDelete
			NoAudit
		}
		column {|
			name | type
			ID | key(bigint)
			Name | text
		}
		run :query {
			from User u
			from UserRole ur eq(u.ID, ur.User)
			from Role r eq(ur.Role, r.ID)
			and eq(arity.CreatedBy, u.ID)
			select u.Username Name, cat(u.FirstName, " ", u.LastName) Fullname
		}
	}

	let EnumTable input :type :db.table {
		context input
		extra :db.column{arity 0+}
		data :db.data{arity 1+}
	}
	let EnumTable output :type :db.table {
		context output
		tag {|
			name
			HardDelete
			NoAudit
		}
		column {|
			name | type
			ID | key(bigint)
			Name | text
		}
		column :query{
			from $input.extra e
			select e.*
		}
		data :query{
			from $input.data d
			select d.*
		}
	}

	let MyEnum1 :EnumTable {
		data {|
			ID | Name
			1 | Spin
			2 | "Round Up"
			3 | "Round Down"
			4 | Dance
		}
	}


*/

type RootValue struct {
	Index      int64
	Identifier string
	Tag        string
	SourceType string
	Type       string

	Fields []Field
}

type Field struct {
	Name  string
	Index int64

	Type     string
	ValueSet []ValueSet
}

type ValueSet struct {
	Key   string
	Index int64

	Value []Value
}

type Value struct {
	Index int64
	Raw   string
	Field *Field
}

type Root struct {
	List []RootValue
}

func run(ctx context.Context) error {
	root := &Root{
		List: []RootValue{
			{
				Index:      1,
				Identifier: "CarMakes",
				Tag:        "tags",
				Type:       "db.table",

				Fields: []Field{
					{
						Name:  "tag",
						Index: 1,
						ValueSet: []ValueSet{
							{
								Index: 1,
								Value: []Value{
									{
										Index: 1,
										Raw:   "HardDelete",
									},
								},
							},
							{
								Index: 2,
								Value: []Value{
									{
										Index: 1,
										Raw:   "NoAudit",
									},
								},
							},
						},
					},
					{
						Name:  "column",
						Index: 2,
						ValueSet: []ValueSet{
							{
								Key:   "Name",
								Index: 1,
								Value: []Value{
									{
										Index: 1,
										Raw:   "ID",
									},
								},
							},
							{
								Key:   "Type",
								Index: 2,
								Value: []Value{
									{
										Index: 1,
										Raw:   "key",
										Field: &Field{
											Index: 1,
											ValueSet: []ValueSet{
												{
													Value: []Value{
														{
															Index: 1,
															Raw:   "bigint",
														},
													},
												},
											},
										},
									},
								},
							},
						},
					},
				},
			},
		},
	}
	_ = root
	return nil
}
