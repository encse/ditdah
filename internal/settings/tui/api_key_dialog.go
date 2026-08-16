package tui

import (
	"morsemanual/internal/tui/components"
	"morsemanual/internal/tui/keybinding"
	"morsemanual/internal/tui/modal"

	"github.com/rivo/tview"
)

type apiKeyDialog struct {
	content    tview.Primitive
	apiKey     components.InputField
	message    components.TextView
	update     components.Button
	cancel     components.Button
	focusables []tview.Primitive
	validate   func(apiKey string) error
	handle     modal.Handle
}

func newAPIKeyDialog(
	controls components.Factory,
	validate func(apiKey string) error,
) *apiKeyDialog {
	controls = controls.Modal()
	dialog := &apiKeyDialog{validate: validate}
	dialog.apiKey = controls.InputField("API key", "")
	dialog.apiKey.SetLabelWidth(settingsLabelWidth)
	dialog.apiKey.SetMaskCharacter('*')
	dialog.message = controls.TextView()
	dialog.message.SetStyle(components.TextViewDanger)
	dialog.message.SetTextAlign(tview.AlignCenter)
	dialog.update = controls.Button("Update")
	dialog.cancel = controls.Button("Cancel")
	dialog.update.SetSelectedFunc(dialog.submit)
	dialog.cancel.SetSelectedFunc(dialog.close)
	dialog.focusables = []tview.Primitive{
		dialog.apiKey,
		dialog.update,
		dialog.cancel,
	}
	dialog.content = dialog.layout(controls)
	return dialog
}

func (d *apiKeyDialog) Content() tview.Primitive { return d.content }

func (d *apiKeyDialog) Focusables() []tview.Primitive { return d.focusables }

func (d *apiKeyDialog) KeyBindings() []keybinding.Binding { return nil }

func (d *apiKeyDialog) Size() modal.Size {
	return modal.Size{Width: 58, Height: 8}
}

func (d *apiKeyDialog) submit() {
	if d.validate != nil {
		if err := d.validate(d.apiKey.Value()); err != nil {
			d.message.SetText("Error: " + err.Error())
			return
		}
	}
	d.close()
}

func (d *apiKeyDialog) close() {
	if d.handle != nil {
		d.handle.Close()
	}
}

func (d *apiKeyDialog) layout(controls components.Factory) tview.Primitive {
	body := tview.NewFlex().
		SetDirection(tview.FlexRow).
		AddItem(d.apiKey, 1, 0, false).
		AddItem(d.message, 1, 0, false).
		AddItem(centeredButtons(d.update, d.cancel), 1, 0, false)
	surface := controls.TextView()
	surface.SetBorder(" QRZ.com API key ")
	return tview.NewPages().
		AddPage("surface", surface, true, true).
		AddPage("content", pad(body, 2, 3), true, true)
}
