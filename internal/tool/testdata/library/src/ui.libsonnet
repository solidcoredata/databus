[
    {
        Name: "app1.coredata.biz/n/ui/book",
        NamePrev: "",
        Type: "solidcoredata.org/t/ui/searchlistdetail",
        Binds: [
            {Alias: "b", Name: "app1.coredata.biz/n/table/book"},
        ],
        Roles: [
            {
                Name: "prop",
                Fields: [
                    {KV: {display: "Books"}},
                ],
            },
            {
                Name: "action",
                Fields: [],
            },
            {
                Name: "schema",
                Fields: [
                    {Alias: "b", KV: {name: "name", display: "Book Title",}},
                    {Alias: "b", KV: {name: "genre", display: "Genre", type: "link", variant: "dropdown"}},
                    {Alias: "b", KV: {name: "page_count", display: "Number of Pages",}},
                ],
            },
        ],
    },
    {
        Name: "app1.coredata.biz/n/ui/genre",
        NamePrev: "",
        Type: "solidcoredata.org/t/ui/searchlistdetail",
        Binds: [
            {Alias: "g", Name: "app1.coredata.biz/n/table/genre"},
        ],
        Roles: [
            {
                Name: "prop",
                Fields: [
                    {KV: {display: "Genres"}},
                ],
            },
            {
                Name: "action",
                Fields: [],
            },
            {
                Name: "schema",
                Fields: [
                    {Alias: "g", KV: {name: "name", display: "Book Title",}},
                ],
            },
        ],
    },
]