package tui

import (
	"fmt"

	"morsemanual/internal/tui/components"
	"morsemanual/internal/tui/keybinding"
	"morsemanual/internal/tui/modal"

	"github.com/rivo/tview"
)

type loginDialog struct {
	modal.Layout
	callsign   components.InputField
	password   components.InputField
	message    components.TextView
	login      components.Button
	cancel     components.Button
	focusables []tview.Primitive
	loginWith  func(string)
	handle     modal.Handle
}

func newLoginDialog(
	controls components.Factory,
	callsign string,
	loginWith func(string),
) *loginDialog {
	controls = controls.Modal()
	dialog := &loginDialog{loginWith: loginWith}
	dialog.callsign = controls.InputField("Callsign", callsign)
	dialog.callsign.SetLabelWidth(settingsLabelWidth)
	dialog.callsign.SetDisabled(true)
	dialog.password = controls.InputField("Password", "")
	dialog.password.SetLabelWidth(settingsLabelWidth)
	dialog.password.SetMaskCharacter('*')
	dialog.message = controls.TextView()
	dialog.message.SetStyle(components.TextViewDanger)
	dialog.message.SetTextAlign(tview.AlignCenter)
	dialog.login = controls.Button("Login")
	dialog.cancel = controls.Button("Cancel")
	dialog.login.SetSelectedFunc(dialog.submit)
	dialog.cancel.SetSelectedFunc(dialog.close)
	dialog.focusables = []tview.Primitive{
		dialog.password,
		dialog.login,
		dialog.cancel,
	}
	dialog.Layout = dialog.layout(controls)
	return dialog
}

func (d *loginDialog) Focusables() []tview.Primitive { return d.focusables }

func (d *loginDialog) KeyBindings() []keybinding.Binding { return nil }

func (d *loginDialog) submit() {
	if d.loginWith != nil {
		d.loginWith(d.password.Value())
	}
}

func (d *loginDialog) finish(err error) {
	if err != nil {
		d.message.SetText("Error: " + err.Error())
		return
	}
	d.close()
}

func (d *loginDialog) close() {
	if d.handle != nil {
		d.handle.Close()
	}
}

func (d *loginDialog) layout(controls components.Factory) modal.Layout {
	fields := controls.Grid().
		SetRows(1, 1, 1).
		SetColumns(0).
		AddItem(d.callsign, 0, 0, 1, 1, 0, 0, false).
		AddItem(d.password, 2, 0, 1, 1, 0, 0, false)
	return modal.NewLayout(controls, fmt.Sprintf(
		" Login to QRZ.com as %s ",
		d.callsign.Value(),
	), 58).
		Row(fields, 3).
		Row(d.message, 1).
		Actions(centeredButtons(controls, d.login, d.cancel))
}
