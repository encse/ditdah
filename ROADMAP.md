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
- [x] Derive every modal dialog's height from declared rows in one themed layout abstraction; use a uniform two-column inset and one-row top gap, center loading pages explicitly, and make the actions row terminal so no content or accidental slack can appear below the buttons.
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
- [x] Own the Settings menu action at application composition level, persist the decoder audio input in the main Settings dialog, and let the decoder subscribe directly to settings changes for the duration of its `Run` lifecycle.
- [x] Show the Settings form without doing I/O or accepting a context in `Open`; load settings and audio devices and validate saved QRZ credentials inside its structured `Run` lifecycle, and run event-triggered saves and checks as owner-scoped background work.
- [x] Edit QRZ login and Logbook API credentials in nested modal dialogs and persist staged changes only when the main settings dialog is confirmed.
- [x] Add a structured footer component that renders contextual information and the currently available keybindings.
- [x] Add a shared `Layout` that arranges the header, active page content, and footer.
- [x] Define a `Page` abstraction that exposes its identity, title, content, and page-level keybindings without owning the shared header or footer.
- [x] Refactor the logbook view into a page, remove its duplicated outer layout, and load its data from the page-owned `Run(ctx)` lifecycle instead of doing I/O or accepting a context in `New`.
- [x] Add a separate Morse decoder page and application-level F1/F2 navigation between it and the logbook.
- [x] Give every page a context-bound `Run` lifecycle; cancel and wait for the active page before hiding it or stopping the terminal.
- [x] Reconcile pages and nested modals through one application-owned layer stack: derive every layer context from the layer below it, bind modal requests to the exact parent stack entry, keep parent layers running, and hide a removed suffix only after all of its `Run` lifecycles return.
- [x] Bind background work and queued UI updates directly to their page or dialog object, including owners covered by child modals; pin both in the owner task group, discard updates after that owner leaves the requested stack, and wait for running UI callbacks during shutdown.
- [x] Share confirmation and message dialogs from the modal package; confirmation closes and invokes its callback only on OK, Cancel and Escape only close, and the owning page starts work and presents operation errors separately.
- [x] Keep trigger, latest-value mailbox, and multi-subscriber broadcaster helpers together in `internal/syncutil`; notifications remain non-blocking and coalesced without starting goroutines.
- [x] Let the decoder and logbook page subscribe directly, for the duration of `Run`, to coalesced store change notifications emitted after successful mutations; do not forward those changes through application callbacks.
- [x] Coordinate page changes through an initialized, latest-value mailbox inside the application run loop; scope each page and mailbox receiver to one local `errgroup` and wait for both before switching.
- [x] Pass the initial page object directly to `Application.Run`; create a fresh page object for every navigation action and use the page or dialog object itself as its lifecycle identity.
- [x] Let the active page contribute its own application-menu items when shown.
- [x] Remove constructor-captured and page-stored contexts from UI actions; run blocking actions through owner-scoped application background work or the owning layer's `Run(ctx)` lifecycle.
- [x] Dispatch input in overlay, focused-control, page, then application order, while leaving native tview bindings with their controls.
- [x] Enforce mouse focus and capture centrally from declared page, modal, application, and overlay focusables so decorative controls cannot steal input.
- [x] Refresh footer hints when the active page, focused control, or overlay changes.
- [x] Test hint composition, focus-sensitive footer content, modal isolation, and the refactored logbook page.
- [x] Extract an application layer that owns the shared layout and overlays, shows fresh page objects, handles navigation, and provides global keybindings.

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

- [x] Publish a responsive static product site through GitHub Pages and generate deterministic SVG screenshots of the real themed logbook, QSO editor, and populated Morse decoder through test-only screenshot fixtures.
- [x] Build cgo-enabled Linux amd64, Windows amd64, macOS Intel, and SIMD-enabled macOS Apple Silicon release archives on native GitHub-hosted runners; fall back to the scalar implementation when the SIMD build is unavailable and publish version tags as GitHub releases.
- [x] Embed the DitDah favicon as the Windows executable icon during release builds.
- [x] Embed the release tag in every published binary and expose it through `ditdah --version`.
- [x] Show the embedded version, developer callsign, project website, and resolved data directory in an application About dialog.
- [x] Store the SQLite logbook in the platform-native user data directory resolved by `adrg/xdg`.
- [x] Implement QRZ synchronization on top of the validated credentials and stored synchronization state, retaining remote log IDs so edited records can be replaced without duplicates.
- [x] Add the initial read-only logbook TUI with table, details, and search.
- [x] Add QSO creation, editing, and confirmed deletion to the logbook TUI.
- [x] Add the Morse user-interface page shell.
- [x] Split the Morse page into a large scrollable decoded-text panel and a reserved right-side panel.
- [x] Connect the selected live audio capture device to the streaming decoder and Morse output panel; coalesce saved input changes through a trigger and restart only the current audio session.
- [x] Add a decoder-page callsign list with `a`/`Enter`/`d` actions and cached QRZ.com details for the selected entry in the lower-right panel.
- [x] Run decoder callsign lookups in a context-bound page worker which is always joined by `Run`; retry the selected callsign on page reactivation without detached goroutines.
- [x] Highlight every substring occurrence of the selected decoder callsign in the decoded log, add a confirmed clear-log action, and follow new output only while the view is already scrolled to the end.
- [x] Require confirmation before deleting a callsign from the decoder list.
- [x] Keep the QSO editor in its own `internal/qsoeditor/tui` package; construct its factory in the application and let both the logbook and Morse-decoder pages open owner-bound editors without calling each other.
- [x] Refresh the QSO editor's QRZ-derived name and QTH through editor-owned background work when its contacted callsign changes and the field is confirmed or loses focus, without repeating lookups for an unchanged callsign.
- [ ] Add further domain stores and migrations as their features are implemented.
