[
    {
        Name: "solidcoredata.org/t/db/database",
        Roles: [
            {
                Name: "prop",
                Properties: [
                    {Name: "name", Type: "text", FieldName: true, Optional: false, Send: false, Recv: false},
                ],
            },
        ],
    },
    {
        Name: "solidcoredata.org/t/db/table",
        Roles: [
            {
                Name: "prop",
                Properties: [
                    {Name: "name", Type: "text", FieldName: true, Optional: false, Send: false, Recv: false},
                    {Name: "database", Type: "node", Optional: false, Send: false, Recv: false},
                ],
            },
            {
                Name: "schema",
                Properties: [
                    {Name: "name", Type: "text", FieldName: true, Optional: false, Send: true, Recv: false}, // Database column name.
                    {Name: "display", Type: "text", Default: "", Optional: false, Send: true, Recv: false}, // Display name to default to when displaying data from this field.
                    {Name: "type", Type: "text", Optional: false, Send: true, Recv: false}, // Type of the database field.
                    {Name: "fk", Type: "node", Optional: true, Send: false, Recv: false},
                    {Name: "length", Type: "int", Default: 0, Optional: true, Send: true, Recv: false}, // Max length in runes (text) or bytes (bytea).
                    {Name: "nullable", Type: "bool", Optional: true, Send: false, Recv: false, Default: "false"}, // True if the column should be nullable.
                    {Name: "key", Type: "bool", Optional: true, Send: false, Recv: false, Default: "false"}, // True if the column should be nullable.
                    {Name: "comment", Type: "text", Default: "", Optional: true, Send: false, Recv: false},
                ],
            },
        ],
    },
    {
        Name: "solidcoredata.org/t/ui/searchlistdetail",
        Roles: [
            {
                Name: "prop",
                Properties: [
                    {Name: "display", Type: "text", Optional: false, Send: false, Recv: false},
                    {Name: "no_edit", Type: "bool", Optional: true, Send: false, Recv: false},
                    {Name: "no_new", Type: "bool", Optional: true, Send: false, Recv: false},
                    {Name: "no_delete", Type: "bool", Optional: true, Send: false, Recv: false},
                ],
            },
            {
                Name: "action",
                Properties: [
                    {Name: "display", Type: "text", Optional: false, Send: false, Recv: false},
                    {Name: "list", Type: "bool", Optional: true, Send: false, Recv: false},
                    {Name: "detail", Type: "bool", Optional: true, Send: false, Recv: false},
                    {Name: "function", Type: "text", Optional: false, Send: false, Recv: false},
                ],
            },
            {
                Name: "schema",
                Properties: [
                    {Name: "name", Type: "text", Optional: false, Send: false, Recv: true}, // Database column name to bind to.
                    {Name: "display", Type: "text", Optional: false, Send: false, Recv: true}, // Field display.
                    {Name: "type", Type: "text", Optional: false, Send: false, Recv: true}, // Type of the ui field.
                    {Name: "variant", Type: "text", Optional: false, Send: false, Recv: false}, // Type varient of the ui field.
                    {Name: "length", Type: "int", Optional: true, Send: false, Recv: true},
                    {Name: "nullable", Type: "bool", Optional: true, Send: false, Recv: false, Default: "false"},
                    {Name: "nullempty", Type: "bool", Optional: true, Send: false, Recv: false, Default: "false"}, // When true, an "empty" value is considered null.
                    {Name: "help", Type: "text", Optional: true, Send: false, Recv: false},
                ],
            },
        ],
    },
]