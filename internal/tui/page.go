package tui

import (
	"context"

	"ditdah/internal/tui/components"
	"ditdah/internal/tui/keybinding"
	"ditdah/internal/tui/modal"

	"github.com/rivo/tview"
)

// BackgroundWork performs lifecycle-bound work away from the UI thread.
type BackgroundWork func(ctx context.Context)

// Owner is one page or dialog lifecycle.
type Owner = modal.Owner

// PageHost exposes the application services available to pages.
type PageHost interface {
	SetFocus(primitive tview.Primitive)
	Refresh()
	// Update schedules a UI change owned by the current lifecycle of owner.
	// The update is ignored if that exact layer has been closed before it runs.
	Update(owner modal.Owner, update func()) bool
	Components() components.Factory
	// OpenModal requests a child layer and returns without blocking the UI
	// event handler. owner identifies the page or dialog currently requesting
	// the child; the application ignores the request if that layer is no longer
	// on top of the stack.
	OpenModal(owner modal.Owner, dialog modal.Dialog) modal.Handle
	// Background starts work under the current lifecycle of owner without
	// blocking the UI event handler. Work uses Update explicitly when it needs
	// to modify the UI.
	Background(owner modal.Owner, work BackgroundWork) bool
}

// Page is independently navigable application content. Shared application
// chrome such as the header and footer is owned by Layout, not by the page.
type Page interface {
	modal.Owner
	ID() string
	Title() string
	Focusables() []tview.Primitive
	KeyBindings() []keybinding.Binding
	MenuItems() []components.MenuItem
	Run(ctx context.Context)
}
