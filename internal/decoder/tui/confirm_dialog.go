package tui

import (
	"morsemanual/internal/tui/components"
	"morsemanual/internal/tui/keybinding"
	"morsemanual/internal/tui/modal"

	"github.com/rivo/tview"
)

type confirmDialog struct {
	modal.Layout
	message components.TextView
	confirm components.Button
	cancel  components.Button
	action  func()
	handle  modal.Handle
}

func newConfirmDialog(
	controls components.Factory,
	title string,
	message string,
	confirmLabel string,
	action func(),
) *confirmDialog {
	controls = controls.Modal()
	dialog := &confirmDialog{action: action}
	dialog.message = controls.TextView()
	dialog.message.SetText(message)
	dialog.message.SetTextAlign(tview.AlignCenter)
	dialog.confirm = controls.DangerButton(confirmLabel)
	dialog.cancel = controls.Button("Cancel")
	dialog.confirm.SetSelectedFunc(dialog.submit)
	dialog.cancel.SetSelectedFunc(dialog.close)
	buttons := controls.Flex(tview.FlexColumn).
		AddItem(nil, 0, 1, false).
		AddItem(dialog.cancel, 12, 0, false).
		AddItem(nil, 2, 0, false).
		AddItem(dialog.confirm, 12, 0, false).
		AddItem(nil, 0, 1, false)
	dialog.Layout = modal.NewLayout(controls, title, 48).
		Row(dialog.message, 1).
		Spacer().
		Actions(buttons)
	return dialog
}

func (d *confirmDialog) Focusables() []tview.Primitive {
	return []tview.Primitive{d.cancel, d.confirm}
}

func (d *confirmDialog) KeyBindings() []keybinding.Binding { return nil }

func (d *confirmDialog) setHandle(handle modal.Handle) { d.handle = handle }

func (d *confirmDialog) submit() {
	if d.action != nil {
		d.action()
	}
	d.close()
}

func (d *confirmDialog) close() {
	if d.handle != nil {
		d.handle.Close()
	}
}
