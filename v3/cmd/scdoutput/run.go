package main

import (
	"context"
	"database/sql"
	"os"

	_ "modernc.org/sqlite"
)

func run(ctx context.Context) error {
	filename := "testing.db"
	db, err := sql.Open("sqlite", filename)
	if err != nil {
		return err
	}
	defer db.Close()
	defer os.Remove(filename)

	_, err = db.ExecContext(ctx, `
create table node (
	id integer primary key,
	parent integer not null references node(id),
	sort_order integer not null,
	name text not null,
	type_name text not null,
	type_link integer null references node(id),
	value text
);
	`)
	if err != nil {
		return err
	}

	/*
		set SimpleSelect query {
			from   books b

		[0]set(key)
			[0]SimpleSelect
			[1]query
			[2](struct)
				[0]from(key)
					[0]books
					[1]b
	*/
	_, err = db.ExecContext(ctx, `
insert into	node values (
	0, 0,
	0,
	'root', root', null, ''
);

insert into	node values (
	1, 0,
	100,
	'set', 'key', null, 'set'
);
`)

	return nil
}
