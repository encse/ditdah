package tui

import (
	"morsemanual/internal/tui/components"
	"morsemanual/internal/tui/keybinding"
	"morsemanual/internal/tui/modal"

	"github.com/rivo/tview"
)

type confirmDialog struct {
	modal.Layout
	message       components.TextView
	detail        components.TextView
	confirm       components.Button
	cancel        components.Button
	confirmAction func()
	handle        modal.Handle
}

func newConfirmDialog(
	controls components.Factory,
	title string,
	message string,
	detail string,
	confirmLabel string,
	confirmAction func(),
) *confirmDialog {
	return newStyledConfirmDialog(
		controls, title, message, detail, confirmLabel, confirmAction, true,
	)
}

func newRegularConfirmDialog(
	controls components.Factory,
	title string,
	message string,
	detail string,
	confirmLabel string,
	confirmAction func(),
) *confirmDialog {
	return newStyledConfirmDialog(
		controls, title, message, detail, confirmLabel, confirmAction, false,
	)
}

func newStyledConfirmDialog(
	controls components.Factory,
	title string,
	message string,
	detail string,
	confirmLabel string,
	confirmAction func(),
	danger bool,
) *confirmDialog {
	controls = controls.Modal()
	dialog := &confirmDialog{confirmAction: confirmAction}
	dialog.message = controls.TextView()
	dialog.message.SetText(message)
	dialog.message.SetTextAlign(tview.AlignCenter)
	dialog.message.SetWrap(true)
	dialog.message.SetWordWrap(true)
	dialog.detail = controls.TextView()
	dialog.detail.SetText(detail)
	dialog.detail.SetStyle(components.TextViewMuted)
	dialog.detail.SetTextAlign(tview.AlignCenter)
	if danger {
		dialog.confirm = controls.DangerButton(confirmLabel)
	} else {
		dialog.confirm = controls.Button(confirmLabel)
	}
	dialog.cancel = controls.Button("Cancel")
	dialog.confirm.SetSelectedFunc(dialog.submit)
	dialog.cancel.SetSelectedFunc(dialog.close)

	buttons := controls.Flex(tview.FlexColumn).
		AddItem(nil, 0, 1, false).
		AddItem(dialog.cancel, 12, 0, false).
		AddItem(nil, 2, 0, false).
		AddItem(dialog.confirm, 12, 0, false).
		AddItem(nil, 0, 1, false)
	dialog.Layout = modal.NewLayout(controls, title, 58).
		Row(dialog.message, 1).
		Row(dialog.detail, 1).
		Actions(buttons)
	return dialog
}

func (d *confirmDialog) Focusables() []tview.Primitive {
	return []tview.Primitive{d.cancel, d.confirm}
}

func (d *confirmDialog) KeyBindings() []keybinding.Binding {
	return nil
}

func (d *confirmDialog) setHandle(handle modal.Handle) {
	d.handle = handle
}

func (d *confirmDialog) submit() {
	d.close()
	if d.confirmAction != nil {
		d.confirmAction()
	}
}

func (d *confirmDialog) close() {
	if d.handle != nil {
		d.handle.Close()
	}
}
