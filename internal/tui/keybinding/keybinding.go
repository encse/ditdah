// Package keybinding keeps keyboard triggers, handlers, and their
// user-visible descriptions together.
package keybinding

import (
	"strings"

	"github.com/gdamore/tcell/v2"
)

// Hint describes a keyboard action for display in the user interface.
type Hint struct {
	Keys        string
	Description string
}

// BindingProvider exposes the keyboard actions currently handled by a control.
type BindingProvider interface {
	KeyBindings() []Binding
}

// ParentBindingBlocker marks a focused control which must receive input before
// page-level or application-level bindings are considered.
type ParentBindingBlocker interface {
	BlocksParentBindings() bool
}

// Stroke identifies one keyboard event accepted by a binding.
type Stroke struct {
	key  tcell.Key
	rune rune
}

// Key creates a stroke for a non-rune key.
func Key(key tcell.Key) Stroke {
	return Stroke{key: key}
}

// Rune creates a stroke for a typed character.
func Rune(character rune) Stroke {
	return Stroke{key: tcell.KeyRune, rune: character}
}

// Binding invokes one callback for its configured keyboard strokes.
type Binding struct {
	strokes     []Stroke
	description string
	handler     func()
}

// On creates a binding accepting any of the supplied strokes.
func On(
	description string,
	handler func(),
	stroke Stroke,
	additional ...Stroke,
) Binding {
	strokes := append([]Stroke{stroke}, additional...)
	return Binding{
		strokes:     strokes,
		description: description,
		handler:     handler,
	}
}

// OnKey creates a binding handled by one non-rune key.
func OnKey(key tcell.Key, description string, handler func()) Binding {
	return On(description, handler, Key(key))
}

// OnRune creates a binding handled by one typed character.
func OnRune(character rune, description string, handler func()) Binding {
	return On(description, handler, Rune(character))
}

// Handle invokes the callback only when the event matches a configured stroke.
func (b Binding) Handle(event *tcell.EventKey) bool {
	if b.handler == nil {
		return false
	}
	for _, stroke := range b.strokes {
		if stroke.matches(event) {
			b.handler()
			return true
		}
	}
	return false
}

// Hint returns the description derived from the binding's keyboard strokes.
func (b Binding) Hint() Hint {
	keys := make([]string, 0, len(b.strokes))
	for _, stroke := range b.strokes {
		keys = append(keys, stroke.name())
	}
	return Hint{Keys: strings.Join(keys, "/"), Description: b.description}
}

func (s Stroke) matches(event *tcell.EventKey) bool {
	return event.Key() == s.key && (s.key != tcell.KeyRune || event.Rune() == s.rune)
}

func (s Stroke) name() string {
	if s.key == tcell.KeyRune {
		if s.rune == ' ' {
			return "Space"
		}
		return string(s.rune)
	}
	if s.key == tcell.KeyBacktab {
		return "Shift+Tab"
	}
	if name, ok := tcell.KeyNames[s.key]; ok {
		return name
	}
	return "Unknown"
}

// Hints returns the footer-visible hints belonging to handled bindings.
func Hints(bindings []Binding) []Hint {
	hints := make([]Hint, 0, len(bindings))
	for _, binding := range bindings {
		hint := binding.Hint()
		if binding.handler == nil || !visibleInFooter(hint) {
			continue
		}
		hints = append(hints, hint)
	}
	return hints
}

func visibleInFooter(hint Hint) bool {
	switch hint.Keys {
	case "", "Enter", "Esc", "Space", "Enter/Space", "Tab", "Shift+Tab":
		return false
	default:
		return true
	}
}

// MergeBindingHints adds the visible hints from bindings. Later hints win.
func MergeBindingHints(hints []Hint, bindings ...Binding) []Hint {
	return MergeHints(hints, Hints(bindings)...)
}

// MergeHints adds or replaces hints by their key label. Later hints win.
func MergeHints(hints []Hint, additional ...Hint) []Hint {
	merged := append([]Hint(nil), hints...)
	for _, hint := range additional {
		replaced := false
		for index := range merged {
			if merged[index].Keys == hint.Keys {
				merged[index] = hint
				replaced = true
				break
			}
		}
		if !replaced {
			merged = append(merged, hint)
		}
	}
	return merged
}
