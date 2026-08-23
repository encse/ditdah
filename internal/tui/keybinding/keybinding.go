// Package keybinding keeps keyboard triggers, handlers, and their
// user-visible descriptions together.
package keybinding

import (
	"strconv"
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

// Action creates an invokable action without a keyboard shortcut.
func Action(description string, handler func()) Binding {
	return Binding{description: description, handler: handler}
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
	for _, stroke := range b.strokes {
		if stroke.matches(event) {
			return b.Invoke()
		}
	}
	return false
}

// Invoke runs the binding action without synthesizing a keyboard event.
func (b Binding) Invoke() bool {
	if b.handler == nil {
		return false
	}
	b.handler()
	return true
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

// Visible returns the handled bindings which are advertised in the UI.
func Visible(bindings []Binding) []Binding {
	visible := make([]Binding, 0, len(bindings))
	for _, binding := range bindings {
		if binding.handler != nil && visibleInFooter(binding.Hint()) {
			visible = append(visible, binding)
		}
	}
	return visible
}

// Merge adds or replaces visible bindings by their key label. Later bindings
// win, matching keyboard dispatch and the existing hint composition rules.
func Merge(bindings []Binding, additional ...Binding) []Binding {
	merged := append([]Binding(nil), bindings...)
	for _, binding := range Visible(additional) {
		key := binding.Hint().Keys
		replaced := false
		for index := range merged {
			if merged[index].Hint().Keys == key {
				merged[index] = binding
				replaced = true
				break
			}
		}
		if !replaced {
			merged = append(merged, binding)
		}
	}
	return merged
}

func visibleInFooter(hint Hint) bool {
	switch hint.Keys {
	case "", "Enter", "Esc", "Space", "Enter/Space", "Tab", "Shift+Tab":
		return false
	default:
		return true
	}
}

// SplitFunctionBindings separates function-key bindings from all other
// visible bindings.
func SplitFunctionBindings(
	bindings []Binding,
) (functionKeys []Binding, other []Binding) {
	for _, binding := range Visible(bindings) {
		if isFunctionKeyHint(binding.Hint().Keys) {
			functionKeys = append(functionKeys, binding)
		} else {
			other = append(other, binding)
		}
	}
	return functionKeys, other
}

func isFunctionKeyHint(keys string) bool {
	parts := strings.Split(keys, "/")
	if len(parts) == 0 {
		return false
	}
	for _, part := range parts {
		if len(part) < 2 || part[0] != 'F' {
			return false
		}
		number, err := strconv.Atoi(part[1:])
		if err != nil || number < 1 || number > 64 {
			return false
		}
	}
	return true
}
