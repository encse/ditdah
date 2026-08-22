package tui

import (
	"strings"

	"ditdah/internal/tui/components"
	"ditdah/internal/tui/keybinding"
	"ditdah/internal/tui/modal"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

type callsignDialog struct {
	modal.Layout
	input   components.InputField
	message components.TextView
	confirm components.Button
	cancel  components.Button
	save    func(string) error
	handle  modal.Handle
}

func newCallsignDialog(
	controls components.Factory,
	title string,
	confirmLabel string,
	value string,
	save func(string) error,
) *callsignDialog {
	controls = controls.Modal()
	dialog := &callsignDialog{save: save}
	dialog.input = controls.InputField("Callsign", value)
	dialog.input.SetLabelWidth(12)
	dialog.message = controls.TextView()
	dialog.message.SetStyle(components.TextViewDanger)
	dialog.confirm = controls.Button(confirmLabel)
	dialog.cancel = controls.Button("Cancel")
	dialog.confirm.SetSelectedFunc(dialog.submit)
	dialog.cancel.SetSelectedFunc(dialog.close)
	dialog.input.SetBindings(keybinding.OnKey(
		tcell.KeyEnter,
		strings.ToLower(confirmLabel)+" callsign",
		dialog.submit,
	))

	buttons := controls.Flex(tview.FlexColumn).
		AddItem(nil, 0, 1, false).
		AddItem(dialog.cancel, 10, 0, false).
		AddItem(nil, 2, 0, false).
		AddItem(dialog.confirm, 10, 0, false).
		AddItem(nil, 0, 1, false)
	dialog.Layout = modal.NewLayout(
		controls,
		" "+title+" ",
		48,
	).
		Row(dialog.input, 1).
		Row(dialog.message, 1).
		Actions(buttons)
	return dialog
}

func (d *callsignDialog) Focusables() []tview.Primitive {
	return []tview.Primitive{d.input, d.cancel, d.confirm}
}

func (d *callsignDialog) KeyBindings() []keybinding.Binding { return nil }

func (d *callsignDialog) setHandle(handle modal.Handle) { d.handle = handle }

func (d *callsignDialog) submit() {
	if d.save != nil {
		if err := d.save(d.input.Value()); err != nil {
			d.message.SetText(err.Error())
			return
		}
	}
	d.close()
}

func (d *callsignDialog) close() {
	if d.handle != nil {
		d.handle.Close()
	}
}
