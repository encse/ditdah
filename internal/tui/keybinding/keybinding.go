// Package keybinding describes keyboard actions and user-visible keyboard
// hints independently from the controls which handle them.
package keybinding

import "github.com/gdamore/tcell/v2"

// Hint describes a keyboard action for display in the user interface.
type Hint struct {
	Keys        string
	Description string
}

// HintProvider exposes the keyboard actions currently available on a control.
// The control may still delegate the actual event handling to tview.
type HintProvider interface {
	KeyHints() []Hint
}

// ParentBindingBlocker marks a focused control which must receive input before
// page-level or application-level bindings are considered.
type ParentBindingBlocker interface {
	BlocksParentBindings() bool
}

// Binding is an application-handled keyboard action and its user-visible hint.
type Binding struct {
	Hint    Hint
	Handler func(event *tcell.EventKey) bool
}

// OnKey creates a binding handled by one non-rune key.
func OnKey(key tcell.Key, hint Hint, handler func()) Binding {
	return Binding{
		Hint: hint,
		Handler: func(event *tcell.EventKey) bool {
			if event.Key() != key {
				return false
			}
			handler()
			return true
		},
	}
}

// Handle invokes the binding. It reports whether the event was handled.
func (b Binding) Handle(event *tcell.EventKey) bool {
	if b.Handler == nil {
		return false
	}
	return b.Handler(event)
}

// Hints returns the user-visible hints belonging to application bindings.
func Hints(bindings []Binding) []Hint {
	hints := make([]Hint, 0, len(bindings))
	for _, binding := range bindings {
		hints = append(hints, binding.Hint)
	}
	return hints
}
