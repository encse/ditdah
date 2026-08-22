// Package modal defines application-level modal dialog contracts.
package modal

import (
	"context"

	"ditdah/internal/tui/keybinding"

	"github.com/rivo/tview"
)

// Size is the preferred terminal size of a modal dialog.
type Size struct {
	Width  int
	Height int
}

// Owner identifies one page or dialog lifecycle.
type Owner interface {
	Content() tview.Primitive
}

// Dialog is self-contained application content displayed modally.
type Dialog interface {
	Owner
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
