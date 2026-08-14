# Roadmap

This document records implementation decisions and future technical work.

## Architecture

- [x] Keep concrete stateful implementations unexported and expose them through small interfaces.
- [x] Pass and return public domain models by value; generated persistence pointers stay inside persistence implementations.
- [x] Represent nullable domain values with value-based `optional.Value[T]`; do not reserve zero values as missing-value sentinels.
- [x] Keep UI, domain models, and persistence separate.
- [x] Keep SQLite connection setup, migrations, and the shared generated database model in `internal/database`.
- [x] Keep domain queries and persistence mapping in their domain package, such as `internal/logbook`.
- [x] Use the standard `*sql.DB` directly; the application owns and closes it, while domain stores use the shared connection without an extra database wrapper interface.
- [x] Use `tview` for the classic pane-, table-, and menu-oriented terminal interface.
- [x] Use the Nord color scheme to match the original Python application.

## Database

- [x] Use the pure-Go `modernc.org/sqlite` driver.
- [x] Store schema changes as versioned Goose migrations in `internal/database/migrations`.
- [x] Generate the shared Jet table and persistence models from the fully migrated schema.
- [x] Use Jet-generated types and query builders instead of handwritten row types and query SQL.
- [x] Keep the application-owned QSO domain model separate from Jet's generated persistence model.
- [x] Store whether and when a QSO was synchronized to QRZ.
- [x] Allow arbitrary operating-mode strings so additional modes do not require a closed enum or database constraint.
- [x] Allow Jet SQL DSL and generated table packages to use dot imports inside SQLite store implementations.

## Future work

- [ ] Implement QRZ synchronization on top of the stored synchronization state.
- [x] Add the initial read-only logbook TUI with table, details, and search.
- [ ] Add QSO creation, editing, and deletion to the logbook TUI.
- [ ] Add the Morse user-interface view.
- [ ] Add further domain stores and migrations as their features are implemented.
