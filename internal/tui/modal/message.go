package modal

import (
	"morsemanual/internal/tui/components"
	"morsemanual/internal/tui/keybinding"

	"github.com/rivo/tview"
)

type messageDialog struct {
	Layout
	ok     components.Button
	handle Handle
}

func OpenError(
	host Opener,
	owner tview.Primitive,
	title string,
	message string,
) Dialog {
	controls := host.Components().Modal()
	dialog := &messageDialog{}
	messageView := controls.TextView()
	messageView.SetText(message)
	messageView.SetStyle(components.TextViewDanger)
	dialog.ok = controls.Button("OK")
	dialog.ok.SetSelectedFunc(dialog.close)
	buttons := controls.Flex(tview.FlexColumn).
		AddItem(nil, 0, 1, false).
		AddItem(dialog.ok, 12, 0, false).
		AddItem(nil, 0, 1, false)
	dialog.Layout = NewLayout(controls, title, 58).
		Row(messageView, 1).
		Actions(buttons)
	dialog.handle = host.OpenModal(owner, dialog)
	return dialog
}

func (d *messageDialog) Focusables() []tview.Primitive {
	return []tview.Primitive{d.ok}
}

func (d *messageDialog) KeyBindings() []keybinding.Binding { return nil }

func (d *messageDialog) close() {
	if d.handle != nil {
		d.handle.Close()
	}
}
