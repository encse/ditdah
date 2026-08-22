package components

import (
	"ditdah/internal/tui/keybinding"

	"github.com/rivo/tview"
)

// Footer is the structured bottom-level application footer. Context is shown
// separately from the keyboard hints available in the current UI state.
type Footer interface {
	tview.Primitive
	SetContext(context string)
	SetKeyBindings(bindings []keybinding.Binding)
}

type footer struct {
	*tview.Flex
	context TextView
	hints   *keyHints
}

func newFooter(controls Factory, theme Theme, changed func()) Footer {
	context := controls.TextView()
	context.SetStyle(TextViewMuted)

	hints := newKeyHints(theme, tview.AlignCenter, changed)

	footer := &footer{
		Flex:    tview.NewFlex(),
		context: context,
		hints:   hints,
	}
	footer.SetContext("")
	return footer
}

func (f *footer) SetContext(context string) {
	f.context.SetText(context)
	f.Clear()
	if context != "" {
		f.AddItem(f.context, 0, 1, false)
		f.AddItem(f.hints, 0, 3, false)
		return
	}
	f.AddItem(f.hints, 0, 1, false)
}

func (f *footer) SetKeyBindings(bindings []keybinding.Binding) {
	f.hints.SetBindings(bindings)
}
