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
- [x] Keep reusable terminal controls in their own `internal/tui/components` package, with unexported implementations behind small interfaces and a shared themed factory.
- [x] Display modal dialogs and control popups through one stackable overlay host, allowing controls inside overlays to open further overlays.
- [ ] Introduce a consistent wrapper component layer over tview/tcell instead of styling raw widgets ad hoc.

## TUI layout and navigation

- [x] Introduce separate keybinding metadata for application-handled bindings and hints for behavior handled natively by tview controls.
- [x] Let reusable controls expose their context-sensitive key hints, including table navigation and input/select actions, without reimplementing tview's event handling.
- [x] Add a structured header component that can contain the current page title, application menu, and status information.
- [x] Add a structured footer component that renders contextual information and the currently available keybindings.
- [ ] Add a shared `Layout` that arranges the header, active page content, and footer.
- [ ] Define a `Page` abstraction that exposes its identity, title, content, and page-level keybindings without owning the shared header or footer.
- [ ] Refactor the logbook view into a page and remove its duplicated outer layout, header, and footer management.
- [ ] Dispatch input in overlay, focused-control, page, then application order, while leaving native tview bindings with their controls.
- [ ] Refresh footer hints when the active page, focused control, or overlay changes.
- [ ] Test hint composition, focus-sensitive footer content, modal isolation, and the refactored logbook page.
- [ ] Extract an application layer that owns the shared layout and overlays, registers pages, handles navigation, and provides global keybindings.

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
