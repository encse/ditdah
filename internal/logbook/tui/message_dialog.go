package tui

import (
	"morsemanual/internal/tui/components"
	"morsemanual/internal/tui/keybinding"
	"morsemanual/internal/tui/modal"

	"github.com/rivo/tview"
)

type messageDialog struct {
	modal.Layout
	message components.TextView
	ok      components.Button
	handle  modal.Handle
}

func newMessageDialog(
	controls components.Factory,
	title string,
	message string,
) *messageDialog {
	controls = controls.Modal()
	dialog := &messageDialog{}
	dialog.message = controls.TextView()
	dialog.message.SetText(message)
	dialog.message.SetStyle(components.TextViewDanger)
	dialog.ok = controls.Button("OK")
	dialog.ok.SetSelectedFunc(dialog.close)
	buttons := controls.Flex(tview.FlexColumn).
		AddItem(nil, 0, 1, false).
		AddItem(dialog.ok, 12, 0, false).
		AddItem(nil, 0, 1, false)
	dialog.Layout = modal.NewLayout(controls, title, 58).
		Row(dialog.message, 1).
		Actions(buttons)
	return dialog
}

func (d *messageDialog) Focusables() []tview.Primitive {
	return []tview.Primitive{d.ok}
}

func (d *messageDialog) KeyBindings() []keybinding.Binding { return nil }

func (d *messageDialog) setHandle(handle modal.Handle) { d.handle = handle }

func (d *messageDialog) close() {
	if d.handle != nil {
		d.handle.Close()
	}
}
