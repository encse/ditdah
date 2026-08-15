package tui

import (
	"morsemanual/internal/tui/components"
	"morsemanual/internal/tui/keybinding"
	"morsemanual/internal/tui/modal"

	"github.com/rivo/tview"
)

// PageHost exposes the application services available to pages.
type PageHost interface {
	SetFocus(primitive tview.Primitive)
	Refresh()
	Components() components.Factory
	OpenModal(dialog modal.Dialog) modal.Handle
}

// Page is independently navigable application content. Shared application
// chrome such as the header and footer is owned by Layout, not by the page.
type Page interface {
	ID() string
	Title() string
	Content() tview.Primitive
	Focusables() []tview.Primitive
	KeyBindings() []keybinding.Binding
}
