// Package modal defines application-level modal dialog contracts.
package modal

import (
	"context"

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
	// Run owns work performed while the dialog is visible and must wait for all
	// child work before returning.
	Run(ctx context.Context)
}

// Handle requests that an open modal dialog and its descendants close.
type Handle interface {
	Close()
}
