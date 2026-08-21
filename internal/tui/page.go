package tui

import (
	"context"

	"morsemanual/internal/tui/components"
	"morsemanual/internal/tui/keybinding"
	"morsemanual/internal/tui/modal"

	"github.com/rivo/tview"
)

// BackgroundWork performs lifecycle-bound work away from the UI thread.
type BackgroundWork func(ctx context.Context)

// PageHost exposes the application services available to pages.
type PageHost interface {
	SetFocus(primitive tview.Primitive)
	Refresh()
	Update(update func())
	Components() components.Factory
	// OpenModal requests a child layer and returns without blocking the UI
	// event handler. owner identifies the page or dialog currently requesting
	// the child; the application ignores the request if that layer is no longer
	// on top of the stack.
	OpenModal(owner tview.Primitive, dialog modal.Dialog) modal.Handle
	// Background starts work under the current lifecycle of owner without
	// blocking the UI event handler. Work uses Update explicitly when it needs
	// to modify the UI.
	Background(owner tview.Primitive, work BackgroundWork) bool
}

// Page is independently navigable application content. Shared application
// chrome such as the header and footer is owned by Layout, not by the page.
type Page interface {
	ID() string
	Title() string
	Content() tview.Primitive
	Focusables() []tview.Primitive
	KeyBindings() []keybinding.Binding
	MenuItems() []components.MenuItem
	SettingsChanged()
	Run(ctx context.Context)
}
