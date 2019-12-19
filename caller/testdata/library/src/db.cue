package bus

db :: [
	{
		Name: "app1.coredata.biz/n/database"
		NameAlt: []
		Type: "solidcoredata.org/t/db/database"
		Binds: []
		Roles: [
			{
				Name: "prop"
				Fields: [
					{KV: {"name": "library"}},
				]
			},
		]
	},
	{
		Name: "app1.coredata.biz/n/table/genre"
		NameAlt: []
		Type: "solidcoredata.org/t/db/table"
		Binds: []
		Roles: [
			{
				Name: "prop"
				Fields: [
					{KV: {name: "genre", database: "app1.coredata.biz/n/database"}},
				]
			},
			{
				Name: "schema"
				Fields: [
					{KV: {name: "id", key:    true, type:     "int"}},
					{KV: {name: "name", type: "text", length: 1000}},
				]
			},
		]
	},
	{
		Name: "app1.coredata.biz/n/table/book"
		NameAlt: []
		Type: "solidcoredata.org/t/db/table"
		Binds: []
		Roles: [
			{
				Name: "prop"
				Fields: [
					{KV: {name: "book", database: "app1.coredata.biz/n/database"}},
				]
			},
			{
				Name: "schema"
				Fields: [
					{KV: {name: "id", type:         "int", key:      true}},
					{KV: {name: "name", type:       "text", length:  1000}},
					{KV: {name: "genre", type:      "int", nullable: true, fk: "app1.coredata.biz/n/table/genre"}},
					{KV: {name: "page_count", type: "int", nullable: true}},
				]
			},
		]
	},
]
