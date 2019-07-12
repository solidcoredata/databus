{
    Root: "memory://verify/output",
    Enteries: [
        {
            Name: "SQL Gen",
            Call: "memory://run/tool/sql",
            Options: {
                variant: "crdb",
            },
        },
    ],
}