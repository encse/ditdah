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
- [x] Add a structured header component for fixed application navigation and status information; intentionally omit the redundant current-page title.
- [x] Add an application header menu with an Exit action backed by the application shutdown path.
- [x] Represent the fixed top navigation as one menu-bar component whose elements are the hamburger menu, F1 Logbook, and F2 Morse decoder; keep it outside contextual hint refreshes.
- [x] Use a uniform two-cell gap around the hamburger and between top navigation actions, and keep the hamburger background unchanged while focused or open.
- [x] Use an ASCII `[=]` menu marker because ambiguous-width Unicode symbols such as `☰` can desynchronize tcell's cursor position from the host terminal.
- [x] Make the header and footer keybinding hints clickable through the same binding handlers used by keyboard input, and retain function-key navigation while the application menu is open.
- [x] Let the overlay host focus each new layer exactly once; avoid re-focusing layers already selected by `tview.Pages`, because `tview.Application.SetFocus` blurs even when the target is unchanged.
- [x] Add a settings modal for the station callsign and validated QRZ credentials, opened from the application menu.
- [x] Add an application menu dialog for selecting and persisting the Morse decoder audio input by device ID.
- [x] Draw a settings progress state before synchronously validating saved QRZ credentials, then reveal the interactive form without fire-and-forget goroutines.
- [x] Edit QRZ login and Logbook API credentials in nested modal dialogs and persist staged changes only when the main settings dialog is confirmed.
- [x] Add a structured footer component that renders contextual information and the currently available keybindings.
- [x] Add a shared `Layout` that arranges the header, active page content, and footer.
- [x] Define a `Page` abstraction that exposes its identity, title, content, and page-level keybindings without owning the shared header or footer.
- [x] Refactor the logbook view into a page and remove its duplicated outer layout, header, and footer management.
- [x] Add a separate Morse decoder page and application-level F1/F2 navigation between it and the logbook.
- [x] Give every page a context-bound `Run` lifecycle; cancel and wait for the active page before hiding it or stopping the terminal.
- [x] Coordinate page changes through an initialized, latest-value mailbox inside the application run loop; scope each page and mailbox receiver to one local `errgroup` and wait for both before switching.
- [x] Pass the initial page ID directly to `Application.Run`; create the page mailbox there so startup and later navigation use the same data path.
- [x] Let pages contribute their own application-menu items when registered.
- [ ] Pass the application runtime context to keybinding handlers during dispatch, then remove constructor-captured and page-stored contexts from UI actions, including context-dependent menu items.
- [x] Dispatch input in overlay, focused-control, page, then application order, while leaving native tview bindings with their controls.
- [x] Enforce mouse focus and capture centrally from declared page, modal, application, and overlay focusables so decorative controls cannot steal input.
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
- [x] Cache positive and negative lookups by normalized input callsign while retaining the provider query callsign, provider metadata, and optional QRZ station data.
- [x] Resolve callsigns through a cache-first service; on a miss, query QRZ.com only when XML login credentials are configured, then persist positive and not-found results.
- [x] Allow arbitrary operating-mode strings so additional modes do not require a closed enum or database constraint.
- [x] Allow Jet SQL DSL and generated table packages to use dot imports inside SQLite store implementations.

## Future work

- [x] Implement QRZ synchronization on top of the validated credentials and stored synchronization state, retaining remote log IDs so edited records can be replaced without duplicates.
- [x] Add the initial read-only logbook TUI with table, details, and search.
- [x] Add QSO creation, editing, and confirmed deletion to the logbook TUI.
- [x] Add the Morse user-interface page shell.
- [x] Split the Morse page into a large scrollable decoded-text panel and a reserved right-side panel.
- [x] Connect the selected live audio capture device to the streaming decoder and Morse output panel; coalesce saved input changes through a trigger and restart only the current audio session.
- [x] Add a decoder-page callsign list with `a`/`Enter`/`d` actions and cached QRZ.com details for the selected entry in the lower-right panel.
- [x] Run decoder callsign lookups in a context-bound page worker which is always joined by `Run`; retry the selected callsign on page reactivation without detached goroutines.
- [x] Highlight every substring occurrence of the selected decoder callsign in the decoded log, add a confirmed clear-log action, and follow new output only while the view is already scrolled to the end.
- [x] Open a prefilled CW QSO editor from Enter on a selected Morse-decoder callsign; keep the cross-page API limited to that callsign and let the logbook dialog resolve the station callsign, timestamp, and available QRZ name and QTH.
- [x] Refresh the QSO editor's QRZ-derived name and QTH when its contacted callsign changes and the field is confirmed or loses focus, without repeating lookups for an unchanged callsign.
- [ ] Add further domain stores and migrations as their features are implemented.
