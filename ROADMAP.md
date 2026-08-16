# Roadmap

This document records implementation decisions and future technical work.

## Architecture

- [x] Keep concrete stateful implementations unexported and expose them through small interfaces.
- [x] Pass and return public domain models by value; generated persistence pointers stay inside persistence implementations.
- [x] Represent nullable domain values with value-based `optional.Value[T]`; do not reserve zero values as missing-value sentinels.
- [x] Keep UI, domain models, and persistence separate.
- [x] Keep SQLite connection setup, migrations, and the shared generated database model in `internal/database`.
- [x] Keep domain queries and persistence mapping in their domain package, such as `internal/logbook`.
- [x] Construct all domain stores once at application startup and pass them through the composition layer as one store collection.
- [x] Use the standard `*sql.DB` directly; the application owns and closes it, while domain stores use the shared connection without an extra database wrapper interface.
- [x] Use `tview` for the classic pane-, table-, and menu-oriented terminal interface.
- [x] Use the Nord color scheme to match the original Python application.
- [x] Keep reusable terminal controls in their own `internal/tui/components` package, with unexported implementations behind small interfaces and a shared themed factory.
- [x] Display modal dialogs and control popups through one stackable overlay host, allowing controls inside overlays to open further overlays.
- [ ] Introduce a consistent wrapper component layer over tview/tcell instead of styling raw widgets ad hoc.

## TUI layout and navigation

- [x] Keep each advertised key description on the same `Binding` as its handler, and derive footer hints exclusively from active handled bindings.
- [x] Derive each binding's key label and event matching from one configured keyboard trigger instead of duplicating key checks in handlers and hints.
- [x] Do not advertise implicit native tview behavior unless the application wraps it in its own binding.
- [x] Keep conventional Enter, Escape, Space, and Tab bindings active but hide them from the footer.
- [x] Handle modal Escape centrally in the application, never in individual modal controls or dialogs.
- [x] Keep Tab focus navigation application-owned; control popups close themselves when that focus move dismisses them.
- [x] Add a structured header component that can contain the current page title, application menu, and status information.
- [x] Add an application header menu with an Exit action backed by the application shutdown path.
- [x] Add a settings modal for the station callsign and validated QRZ credentials, opened from the application menu.
- [x] Validate saved QRZ login and Logbook API credentials when settings open, and edit them in nested modal dialogs.
- [x] Add a structured footer component that renders contextual information and the currently available keybindings.
- [x] Add a shared `Layout` that arranges the header, active page content, and footer.
- [x] Define a `Page` abstraction that exposes its identity, title, content, and page-level keybindings without owning the shared header or footer.
- [x] Refactor the logbook view into a page and remove its duplicated outer layout, header, and footer management.
- [x] Dispatch input in overlay, focused-control, page, then application order, while leaving native tview bindings with their controls.
- [x] Refresh footer hints when the active page, focused control, or overlay changes.
- [x] Test hint composition, focus-sensitive footer content, modal isolation, and the refactored logbook page.
- [x] Extract an application layer that owns the shared layout and overlays, registers pages, handles navigation, and provides global keybindings.

## Database

- [x] Use the pure-Go `modernc.org/sqlite` driver.
- [x] Store schema changes as versioned Goose migrations in `internal/database/migrations`.
- [x] Generate the shared Jet table and persistence models from the fully migrated schema.
- [x] Use Jet-generated types and query builders instead of handwritten row types and query SQL.
- [x] Keep the application-owned QSO domain model separate from Jet's generated persistence model.
- [x] Store whether and when a QSO was synchronized to QRZ.
- [x] Persist the station callsign and QRZ credentials in a singleton application settings record.
- [x] Allow arbitrary operating-mode strings so additional modes do not require a closed enum or database constraint.
- [x] Allow Jet SQL DSL and generated table packages to use dot imports inside SQLite store implementations.

## Future work

- [ ] Implement QRZ synchronization on top of the validated credentials and stored synchronization state.
- [x] Add the initial read-only logbook TUI with table, details, and search.
- [x] Add QSO creation, editing, and confirmed deletion to the logbook TUI.
- [ ] Add the Morse user-interface view.
- [ ] Add further domain stores and migrations as their features are implemented.
