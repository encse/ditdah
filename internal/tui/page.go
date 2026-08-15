package tui

import (
	"morsemanual/internal/tui/keybinding"

	"github.com/rivo/tview"
)

// Page is independently navigable application content. Shared application
// chrome such as the header and footer is owned by Layout, not by the page.
type Page interface {
	ID() string
	Title() string
	Content() tview.Primitive
	KeyBindings() []keybinding.Binding
}
