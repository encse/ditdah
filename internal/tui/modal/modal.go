// Package modal defines application-level modal dialog contracts.
package modal

import (
	"morsemanual/internal/tui/keybinding"

	"github.com/rivo/tview"
)

// Size is the preferred terminal size of a modal dialog.
type Size struct {
	Width  int
	Height int
}

// Dialog is self-contained application content displayed modally.
type Dialog interface {
	Content() tview.Primitive
	Focusables() []tview.Primitive
	KeyBindings() []keybinding.Binding
	Size() Size
}

// Handle closes an open modal dialog.
type Handle interface {
	Close()
}
