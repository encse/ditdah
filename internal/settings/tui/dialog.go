// Package tui implements the settings terminal dialog.
package tui

import (
	"context"
	"fmt"

	domain "morsemanual/internal/settings"
	ui "morsemanual/internal/tui"
	"morsemanual/internal/tui/components"
	"morsemanual/internal/tui/keybinding"
	"morsemanual/internal/tui/modal"

	"github.com/rivo/tview"
)

const settingsLabelWidth = 20

type saveSettingsFunc func(domain.Settings) (domain.Settings, error)

type dialog struct {
	content tview.Primitive
	values  domain.Settings
	save    saveSettingsFunc

	stationCallsign components.InputField
	qrzPassword     components.InputField
	qrzAPIKey       components.InputField
	message         components.TextView
	ok              components.Button
	cancel          components.Button
	focusables      []tview.Primitive
	handle          modal.Handle
}

// Open loads the current settings and displays their modal editor.
func Open(ctx context.Context, host ui.PageHost, store domain.Store) {
	values, loadErr := store.Load(ctx)
	dialog := newDialog(
		host.Components(),
		values,
		func(values domain.Settings) (domain.Settings, error) {
			return store.Save(ctx, values)
		},
	)
	if loadErr != nil {
		dialog.showError(fmt.Errorf("load settings: %w", loadErr))
	}
	dialog.handle = host.OpenModal(dialog)
}

func newDialog(
	controls components.Factory,
	values domain.Settings,
	save saveSettingsFunc,
) *dialog {
	controls = controls.Modal()
	dialog := &dialog{values: values, save: save}
	dialog.stationCallsign = dialog.input(
		controls,
		"My callsign",
		values.StationCallsign,
	)
	dialog.qrzPassword = dialog.input(
		controls,
		"QRZ.com password",
		values.QRZPassword,
	)
	dialog.qrzPassword.SetMaskCharacter('*')
	dialog.qrzAPIKey = dialog.input(
		controls,
		"QRZ.com API key",
		values.QRZAPIKey,
	)
	dialog.qrzAPIKey.SetMaskCharacter('*')
	dialog.message = controls.TextView()
	dialog.message.SetStyle(components.TextViewDanger)
	dialog.message.SetTextAlign(tview.AlignCenter)
	dialog.ok = controls.Button("OK")
	dialog.cancel = controls.Button("Cancel")
	dialog.ok.SetSelectedFunc(dialog.submit)
	dialog.cancel.SetSelectedFunc(dialog.close)
	dialog.focusables = []tview.Primitive{
		dialog.stationCallsign,
		dialog.qrzPassword,
		dialog.qrzAPIKey,
		dialog.ok,
		dialog.cancel,
	}
	dialog.content = dialog.layout(controls)
	return dialog
}

func (d *dialog) Content() tview.Primitive {
	return d.content
}

func (d *dialog) Focusables() []tview.Primitive {
	return d.focusables
}

func (d *dialog) KeyBindings() []keybinding.Binding {
	return nil
}

func (d *dialog) Size() modal.Size {
	return modal.Size{Width: 64, Height: 11}
}

func (d *dialog) input(
	controls components.Factory,
	label string,
	value string,
) components.InputField {
	input := controls.InputField(label, value)
	input.SetLabelWidth(settingsLabelWidth)
	return input
}

func (d *dialog) submit() {
	values := domain.Settings{
		StationCallsign: d.stationCallsign.Value(),
		QRZPassword:     d.qrzPassword.Value(),
		QRZAPIKey:       d.qrzAPIKey.Value(),
	}
	if d.save != nil {
		saved, err := d.save(values)
		if err != nil {
			d.showError(err)
			return
		}
		values = saved
	}
	d.values = values
	d.close()
}

func (d *dialog) showError(err error) {
	d.message.SetText("Error: " + err.Error())
}

func (d *dialog) close() {
	if d.handle != nil {
		d.handle.Close()
	}
}

func (d *dialog) layout(controls components.Factory) tview.Primitive {
	fields := tview.NewGrid().
		SetRows(1, 1, 1, 1, 1).
		SetColumns(0).
		AddItem(d.stationCallsign, 0, 0, 1, 1, 0, 0, false).
		AddItem(d.qrzPassword, 2, 0, 1, 1, 0, 0, false).
		AddItem(d.qrzAPIKey, 4, 0, 1, 1, 0, 0, false)
	buttons := tview.NewFlex().
		AddItem(nil, 0, 1, false).
		AddItem(d.ok, 12, 0, false).
		AddItem(nil, 2, 0, false).
		AddItem(d.cancel, 12, 0, false).
		AddItem(nil, 0, 1, false)
	body := tview.NewFlex().
		SetDirection(tview.FlexRow).
		AddItem(fields, 5, 0, false).
		AddItem(d.message, 1, 0, false).
		AddItem(buttons, 1, 0, false)
	padded := tview.NewFlex().
		SetDirection(tview.FlexRow).
		AddItem(nil, 2, 0, false).
		AddItem(
			tview.NewFlex().
				AddItem(nil, 3, 0, false).
				AddItem(body, 0, 1, false).
				AddItem(nil, 3, 0, false),
			0,
			1,
			false,
		).
		AddItem(nil, 2, 0, false)
	surface := controls.TextView()
	surface.SetBorder(" Settings ")
	return tview.NewPages().
		AddPage("surface", surface, true, true).
		AddPage("content", padded, true, true)
}
