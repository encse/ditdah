package tui

import (
	"fmt"

	"morsemanual/internal/tui/components"
	"morsemanual/internal/tui/keybinding"
	"morsemanual/internal/tui/modal"

	"github.com/rivo/tview"
)

type loginDialog struct {
	content    tview.Primitive
	callsign   components.InputField
	password   components.InputField
	message    components.TextView
	login      components.Button
	cancel     components.Button
	focusables []tview.Primitive
	validate   func(password string) error
	handle     modal.Handle
}

func newLoginDialog(
	controls components.Factory,
	callsign string,
	validate func(password string) error,
) *loginDialog {
	controls = controls.Modal()
	dialog := &loginDialog{validate: validate}
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
	dialog.content = dialog.layout(controls)
	return dialog
}

func (d *loginDialog) Content() tview.Primitive { return d.content }

func (d *loginDialog) Focusables() []tview.Primitive { return d.focusables }

func (d *loginDialog) KeyBindings() []keybinding.Binding { return nil }

func (d *loginDialog) Size() modal.Size {
	return modal.Size{Width: 58, Height: 10}
}

func (d *loginDialog) submit() {
	if d.validate != nil {
		if err := d.validate(d.password.Value()); err != nil {
			d.message.SetText("Error: " + err.Error())
			return
		}
	}
	d.close()
}

func (d *loginDialog) close() {
	if d.handle != nil {
		d.handle.Close()
	}
}

func (d *loginDialog) layout(controls components.Factory) tview.Primitive {
	fields := controls.Grid().
		SetRows(1, 1, 1).
		SetColumns(0).
		AddItem(d.callsign, 0, 0, 1, 1, 0, 0, false).
		AddItem(d.password, 2, 0, 1, 1, 0, 0, false)
	body := controls.Flex(tview.FlexRow).
		AddItem(fields, 3, 0, false).
		AddItem(d.message, 1, 0, false).
		AddItem(centeredButtons(controls, d.login, d.cancel), 1, 0, false)
	stack := controls.PageStack(fmt.Sprintf(
		" Login to QRZ.com as %s ",
		d.callsign.Value(),
	))
	stack.Add("content", pad(controls, body, 1, 2, 3), true)
	return stack
}
