package tui

import (
	"context"
	"errors"

	"morsemanual/internal/audio"
	domain "morsemanual/internal/settings"
	ui "morsemanual/internal/tui"
	"morsemanual/internal/tui/components"
	"morsemanual/internal/tui/keybinding"
	"morsemanual/internal/tui/modal"

	"github.com/rivo/tview"
)

const morseInputLabelWidth = 14

type morseInputDialog struct {
	modal.Layout
	ctx        context.Context
	store      domain.Store
	values     domain.Settings
	devices    []audio.Device
	handle     modal.Handle
	input      components.SelectField
	message    components.TextView
	ok         components.Button
	cancel     components.Button
	focusables []tview.Primitive
	onSaved    func()
}

// OpenMorseInput displays the audio capture device used by the Morse decoder.
func OpenMorseInput(
	ctx context.Context,
	host ui.PageHost,
	inputs audio.DeviceLister,
	store domain.Store,
	onSaved func(),
) {
	values, loadErr := store.Load(ctx)
	var devices []audio.Device
	var devicesErr error
	if inputs == nil {
		devicesErr = errors.New("audio input is unavailable")
	} else {
		devices, devicesErr = inputs.Devices()
	}
	dialog := newMorseInputDialog(ctx, host, store, values, devices, onSaved)
	dialog.handle = host.OpenModal(dialog)
	dialog.showInitialError(errors.Join(loadErr, devicesErr))
}

func newMorseInputDialog(
	ctx context.Context,
	host ui.PageHost,
	store domain.Store,
	values domain.Settings,
	devices []audio.Device,
	onSaved func(),
) *morseInputDialog {
	controls := host.Components().Modal()
	options := make([]string, len(devices))
	for index, device := range devices {
		options[index] = device.Name
		if device.IsDefault {
			options[index] += " (default)"
		}
	}

	dialog := &morseInputDialog{
		ctx:     ctx,
		store:   store,
		values:  values,
		devices: append([]audio.Device(nil), devices...),
		onSaved: onSaved,
	}
	dialog.input = controls.SelectField(
		"Audio input",
		options,
		selectedMorseInput(values.MorseInputDeviceID, devices),
		morseInputLabelWidth,
		0,
	)
	dialog.message = controls.TextView()
	dialog.message.SetStyle(components.TextViewDanger)
	dialog.message.SetTextAlign(tview.AlignCenter)
	dialog.ok = controls.Button("OK")
	dialog.cancel = controls.Button("Cancel")
	dialog.ok.SetSelectedFunc(dialog.submit)
	dialog.cancel.SetSelectedFunc(dialog.close)
	dialog.focusables = []tview.Primitive{dialog.input, dialog.ok, dialog.cancel}
	dialog.Layout = dialog.layout(controls)
	return dialog
}

func (d *morseInputDialog) Focusables() []tview.Primitive { return d.focusables }

func (d *morseInputDialog) KeyBindings() []keybinding.Binding { return nil }

func (d *morseInputDialog) submit() {
	index, _ := d.input.CurrentOption()
	if index < 0 || index >= len(d.devices) {
		d.showError(errors.New("select an available audio input"))
		return
	}
	values := d.values
	values.MorseInputDeviceID = d.devices[index].ID
	saved, err := d.store.Save(d.ctx, values)
	if err != nil {
		d.showError(err)
		return
	}
	d.values = saved
	if d.onSaved != nil {
		d.onSaved()
	}
	d.close()
}

func (d *morseInputDialog) showInitialError(err error) {
	if err != nil {
		d.showError(err)
		return
	}
	if len(d.devices) == 0 {
		d.showError(errors.New("no audio input devices found"))
		return
	}
	if d.values.MorseInputDeviceID != "" {
		index, _ := d.input.CurrentOption()
		if index < 0 {
			d.showError(errors.New("the selected audio input is unavailable"))
		}
	}
}

func (d *morseInputDialog) showError(err error) {
	d.message.SetText("Error: " + err.Error())
}

func (d *morseInputDialog) close() {
	if d.handle != nil {
		d.handle.Close()
	}
}

func (d *morseInputDialog) layout(controls components.Factory) modal.Layout {
	buttons := centeredButtons(controls, d.ok, d.cancel)
	return modal.NewLayout(controls, " Morse input ", 72).
		Row(d.input, 1).
		Spacer().
		Row(d.message, 1).
		Spacer().
		Actions(buttons)
}

func selectedMorseInput(id string, devices []audio.Device) int {
	if id != "" {
		for index, device := range devices {
			if device.ID == id {
				return index
			}
		}
		return -1
	}
	for index, device := range devices {
		if device.IsDefault {
			return index
		}
	}
	if len(devices) > 0 {
		return 0
	}
	return -1
}
