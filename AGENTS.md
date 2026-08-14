# Project conventions

- Keep concrete stateful implementations unexported and expose them through small interfaces.
- Constructors for those implementations return the interface, usually together with an error.
- Prefer passing and returning domain structs by value in public APIs. Do not expose implementation pointers unless an external API makes that unavoidable.
- Keep UI concerns separate from domain models and persistence.
- The logbook must support multiple operating modes. Do not encode the currently common modes as a closed database constraint or Go enum.
- Keep the logbook schema in `internal/logbook/schema.sql` and use the generated Jet table and persistence-model types instead of handwritten SQL or row structs.
- After changing the logbook schema, run `go generate ./internal/logbook` and commit the refreshed `dbgen` files.
- Dot imports are allowed in the SQLite store only for the Jet SQL DSL and its generated table package.
