package components

import (
	"fmt"
	"strings"

	"morsemanual/internal/tui/keybinding"

	"github.com/rivo/tview"
)

// Footer is the structured bottom-level application footer. Context is shown
// separately from the keyboard hints available in the current UI state.
type Footer interface {
	tview.Primitive
	SetContext(context string)
	SetKeyHints(hints []keybinding.Hint)
}

type footer struct {
	*tview.Flex
	context TextView
	hints   TextView
}

func newFooter(controls Factory) Footer {
	context := controls.TextView()
	context.SetStyle(TextViewMuted)

	hints := controls.TextView()
	hints.SetDynamicColors(true)
	hints.SetTextAlign(tview.AlignCenter)
	hints.SetStyle(TextViewMuted)

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

func (f *footer) SetKeyHints(hints []keybinding.Hint) {
	parts := make([]string, 0, len(hints))
	for _, hint := range hints {
		parts = append(parts, fmt.Sprintf(
			"[::b]%s[-:-:-] %s",
			tview.Escape(hint.Keys),
			tview.Escape(hint.Description),
		))
	}
	f.hints.SetText(strings.Join(parts, "   "))
}
