package tui

import (
	"morsemanual/internal/tui/components"
	"morsemanual/internal/tui/keybinding"
	"morsemanual/internal/tui/modal"

	"github.com/rivo/tview"
)

type confirmDialog struct {
	content tview.Primitive
	message components.TextView
	detail  components.TextView
	confirm components.Button
	cancel  components.Button
	action  func() error
	handle  modal.Handle
}

func newConfirmDialog(
	controls components.Factory,
	title string,
	message string,
	detail string,
	confirmLabel string,
	action func() error,
) *confirmDialog {
	controls = controls.Modal()
	dialog := &confirmDialog{action: action}
	dialog.message = controls.TextView()
	dialog.message.SetText(message)
	dialog.message.SetTextAlign(tview.AlignCenter)
	dialog.message.SetWrap(true)
	dialog.message.SetWordWrap(true)
	dialog.detail = controls.TextView()
	dialog.detail.SetText(detail)
	dialog.detail.SetStyle(components.TextViewMuted)
	dialog.detail.SetTextAlign(tview.AlignCenter)
	dialog.confirm = controls.DangerButton(confirmLabel)
	dialog.cancel = controls.Button("Cancel")
	dialog.confirm.SetSelectedFunc(dialog.submit)
	dialog.cancel.SetSelectedFunc(dialog.close)

	buttons := tview.NewFlex().
		AddItem(nil, 0, 1, false).
		AddItem(dialog.cancel, 12, 0, false).
		AddItem(nil, 2, 0, false).
		AddItem(dialog.confirm, 12, 0, false).
		AddItem(nil, 0, 1, false)
	body := tview.NewFlex().
		SetDirection(tview.FlexRow).
		AddItem(dialog.message, 1, 0, false).
		AddItem(dialog.detail, 1, 0, false).
		AddItem(buttons, 1, 0, false)
	content := tview.NewFlex().
		AddItem(nil, 3, 0, false).
		AddItem(body, 0, 1, false).
		AddItem(nil, 3, 0, false)
	padded := tview.NewFlex().
		SetDirection(tview.FlexRow).
		AddItem(nil, 2, 0, false).
		AddItem(content, 0, 1, false).
		AddItem(nil, 2, 0, false)
	surface := controls.TextView()
	surface.SetBorder(title)
	dialog.content = tview.NewPages().
		AddPage("surface", surface, true, true).
		AddPage("content", padded, true, true)
	return dialog
}

func (d *confirmDialog) Content() tview.Primitive {
	return d.content
}

func (d *confirmDialog) Focusables() []tview.Primitive {
	return []tview.Primitive{d.cancel, d.confirm}
}

func (d *confirmDialog) KeyBindings() []keybinding.Binding {
	return nil
}

func (d *confirmDialog) Size() modal.Size {
	return modal.Size{Width: 58, Height: 7}
}

func (d *confirmDialog) setHandle(handle modal.Handle) {
	d.handle = handle
}

func (d *confirmDialog) submit() {
	if d.action != nil {
		if err := d.action(); err != nil {
			d.detail.SetStyle(components.TextViewDanger)
			d.detail.SetText("Error: " + err.Error())
			return
		}
	}
	d.close()
}

func (d *confirmDialog) close() {
	if d.handle != nil {
		d.handle.Close()
	}
}
