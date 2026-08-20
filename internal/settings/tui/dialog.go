// Package tui implements the settings terminal dialogs.
package tui

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"morsemanual/internal/audio"
	"morsemanual/internal/qrz"
	domain "morsemanual/internal/settings"
	ui "morsemanual/internal/tui"
	"morsemanual/internal/tui/components"
	"morsemanual/internal/tui/keybinding"
	"morsemanual/internal/tui/modal"

	"github.com/rivo/tview"
)

const settingsLabelWidth = 20

type dialog struct {
	modal.Layout
	ctx             context.Context
	host            ui.PageHost
	store           domain.Store
	qrz             qrz.Service
	pages           components.PageStack
	values          domain.Settings
	handle          modal.Handle
	checking        bool
	devices         []audio.Device
	onChanged       func()
	persistedValues domain.Settings

	stationCallsign components.InputField
	morseInput      components.SelectField
	loginStatus     components.TextView
	apiKeyStatus    components.TextView
	message         components.TextView
	login           components.Button
	clearLogin      components.Button
	updateAPIKey    components.Button
	clearAPIKey     components.Button
	ok              components.Button
	cancel          components.Button
	focusables      []tview.Primitive
}

// Open loads and verifies the current settings, then displays their editor.
func Open(
	ctx context.Context,
	host ui.PageHost,
	store domain.Store,
	qrzService qrz.Service,
	inputs audio.DeviceLister,
	onChanged func(),
) {
	values, loadErr := store.Load(ctx)
	var devices []audio.Device
	var devicesErr error
	if inputs == nil {
		devicesErr = errors.New("audio input is unavailable")
	} else {
		devices, devicesErr = inputs.Devices()
	}
	dialog := newDialog(
		ctx, host, store, qrzService, values, devices, onChanged,
	)
	dialog.handle = host.OpenModal(dialog)
	dialog.finishValidation(loadErr)
	if devicesErr != nil {
		dialog.showError(errors.Join(loadErr, devicesErr))
	} else if len(devices) == 0 {
		dialog.showError(errors.New("no audio input devices found"))
	} else if values.MorseInputDeviceID != "" {
		index, _ := dialog.morseInput.CurrentOption()
		if index < 0 {
			dialog.showError(errors.New("the selected audio input is unavailable"))
		}
	}
}

func newDialog(
	ctx context.Context,
	host ui.PageHost,
	store domain.Store,
	qrzService qrz.Service,
	values domain.Settings,
	devices []audio.Device,
	onChanged func(),
) *dialog {
	controls := host.Components().Modal()
	dialog := &dialog{
		ctx:             ctx,
		host:            host,
		store:           store,
		qrz:             qrzService,
		values:          values,
		checking:        true,
		devices:         append([]audio.Device(nil), devices...),
		onChanged:       onChanged,
		persistedValues: values,
	}
	dialog.stationCallsign = dialog.input(
		controls,
		"My callsign",
		values.StationCallsign,
	)
	options := make([]string, len(devices))
	for index, device := range devices {
		options[index] = device.Name
		if device.IsDefault {
			options[index] += " (default)"
		}
	}
	dialog.morseInput = controls.SelectField(
		"Audio input",
		options,
		selectedMorseInput(values.MorseInputDeviceID, devices),
		settingsLabelWidth,
		0,
	)
	dialog.loginStatus = controls.TextView()
	dialog.apiKeyStatus = controls.TextView()
	dialog.message = controls.TextView()
	dialog.message.SetStyle(components.TextViewDanger)
	dialog.message.SetTextAlign(tview.AlignCenter)
	dialog.login = controls.Button("Login")
	dialog.clearLogin = controls.Button("Clear")
	dialog.updateAPIKey = controls.Button("Update")
	dialog.clearAPIKey = controls.Button("Clear")
	dialog.ok = controls.Button("OK")
	dialog.cancel = controls.Button("Cancel")
	dialog.login.SetSelectedFunc(dialog.openLogin)
	dialog.clearLogin.SetSelectedFunc(dialog.clearQRZLogin)
	dialog.updateAPIKey.SetSelectedFunc(dialog.openAPIKey)
	dialog.clearAPIKey.SetSelectedFunc(dialog.clearQRZAPIKey)
	dialog.ok.SetSelectedFunc(dialog.submit)
	dialog.cancel.SetSelectedFunc(dialog.close)
	dialog.focusables = []tview.Primitive{
		dialog.stationCallsign,
		dialog.morseInput,
		dialog.login,
		dialog.clearLogin,
		dialog.updateAPIKey,
		dialog.clearAPIKey,
		dialog.ok,
		dialog.cancel,
	}
	dialog.Layout = dialog.layout(controls)
	dialog.showLoginStatus(nil)
	dialog.showAPIKeyStatus(nil)
	return dialog
}

func (d *dialog) Focusables() []tview.Primitive {
	if d.checking {
		return nil
	}
	return d.focusables
}

func (d *dialog) KeyBindings() []keybinding.Binding {
	return nil
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

func (d *dialog) finishValidation(loadErr error) {
	var loginErr, apiKeyErr error
	if loadErr == nil {
		loginErr, apiKeyErr = d.validateStoredCredentials()
	}
	if loadErr != nil {
		d.showError(fmt.Errorf("load settings: %w", loadErr))
	} else {
		d.showLoginStatus(loginErr)
		d.showAPIKeyStatus(apiKeyErr)
	}
	d.checking = false
	d.pages.Show("settings")
	d.host.SetFocus(d.stationCallsign)
}

func (d *dialog) validateStoredCredentials() (error, error) {
	var loginErr, apiKeyErr error
	if d.values.QRZPassword != "" {
		loginErr = d.qrz.ValidateLogin(
			d.ctx,
			d.values.StationCallsign,
			d.values.QRZPassword,
		)
	}
	if d.values.QRZAPIKey != "" {
		apiKeyErr = d.qrz.ValidateAPIKey(d.ctx, d.values.QRZAPIKey)
	}
	return loginErr, apiKeyErr
}

func (d *dialog) submit() {
	values := d.currentValues()
	if err := d.save(values); err != nil {
		d.showError(err)
		return
	}
	d.close()
}

func (d *dialog) openLogin() {
	callsign := strings.TrimSpace(d.stationCallsign.Value())
	if callsign == "" {
		d.showError(fmt.Errorf("callsign is required before QRZ.com login"))
		return
	}
	login := newLoginDialog(
		d.host.Components(),
		callsign,
		func(password string) error {
			if err := d.qrz.ValidateLogin(d.ctx, callsign, password); err != nil {
				return err
			}
			values := d.currentValues()
			values.StationCallsign = callsign
			values.QRZPassword = password
			d.stage(values)
			d.stationCallsign.SetValue(callsign)
			d.showLoginStatus(nil)
			return nil
		},
	)
	login.handle = d.host.OpenModal(login)
}

func (d *dialog) openAPIKey() {
	editor := newAPIKeyDialog(
		d.host.Components(),
		func(apiKey string) error {
			if err := d.qrz.ValidateAPIKey(d.ctx, apiKey); err != nil {
				return err
			}
			values := d.currentValues()
			values.QRZAPIKey = apiKey
			d.stage(values)
			d.showAPIKeyStatus(nil)
			return nil
		},
	)
	editor.handle = d.host.OpenModal(editor)
}

func (d *dialog) clearQRZLogin() {
	values := d.currentValues()
	values.QRZPassword = ""
	d.stage(values)
	d.showLoginStatus(nil)
}

func (d *dialog) clearQRZAPIKey() {
	values := d.currentValues()
	values.QRZAPIKey = ""
	d.stage(values)
	d.showAPIKeyStatus(nil)
}

func (d *dialog) currentValues() domain.Settings {
	values := d.values
	values.StationCallsign = strings.TrimSpace(d.stationCallsign.Value())
	index, _ := d.morseInput.CurrentOption()
	if index >= 0 && index < len(d.devices) {
		values.MorseInputDeviceID = d.devices[index].ID
	}
	return values
}

func (d *dialog) save(values domain.Settings) error {
	saved, err := d.store.Save(d.ctx, values)
	if err != nil {
		return err
	}
	d.values = saved
	d.message.SetText("")
	if saved != d.persistedValues && d.onChanged != nil {
		d.onChanged()
	}
	d.persistedValues = saved
	return nil
}

func (d *dialog) stage(values domain.Settings) {
	d.values = values
	d.message.SetText("")
}

func (d *dialog) showLoginStatus(validationErr error) {
	switch {
	case d.values.QRZPassword == "":
		d.loginStatus.SetStyle(components.TextViewMuted)
		d.loginStatus.SetText("Not connected")
	case validationErr != nil:
		d.loginStatus.SetStyle(components.TextViewDanger)
		d.loginStatus.SetText("Check failed: " + validationErr.Error())
	default:
		d.loginStatus.SetStyle(components.TextViewAccent)
		d.loginStatus.SetText("Connected")
	}
}

func (d *dialog) showAPIKeyStatus(validationErr error) {
	switch {
	case d.values.QRZAPIKey == "":
		d.apiKeyStatus.SetStyle(components.TextViewMuted)
		d.apiKeyStatus.SetText("Not set")
	case validationErr != nil:
		d.apiKeyStatus.SetStyle(components.TextViewDanger)
		d.apiKeyStatus.SetText("Check failed: " + validationErr.Error())
	default:
		d.apiKeyStatus.SetStyle(components.TextViewAccent)
		d.apiKeyStatus.SetText("Set")
	}
}

func (d *dialog) showError(err error) {
	d.message.SetText("Error: " + err.Error())
}

func (d *dialog) close() {
	if d.handle != nil {
		d.handle.Close()
	}
}

func (d *dialog) layout(controls components.Factory) modal.Layout {
	loginRow := credentialRow(
		controls,
		"QRZ.com",
		d.loginStatus,
		d.login,
		d.clearLogin,
	)
	apiKeyRow := credentialRow(
		controls,
		"QRZ.com API key",
		d.apiKeyStatus,
		d.updateAPIKey,
		d.clearAPIKey,
	)
	fields := controls.Grid().
		SetRows(1, 1, 1, 1, 1, 1, 1).
		SetColumns(0).
		AddItem(d.stationCallsign, 0, 0, 1, 1, 0, 0, false).
		AddItem(d.morseInput, 2, 0, 1, 1, 0, 0, false).
		AddItem(loginRow, 4, 0, 1, 1, 0, 0, false).
		AddItem(apiKeyRow, 6, 0, 1, 1, 0, 0, false)
	buttons := centeredButtons(controls, d.ok, d.cancel)
	progress := controls.TextView()
	progress.SetStyle(components.TextViewAccent)
	progress.SetTextAlign(tview.AlignCenter)
	progress.SetText("Checking QRZ.com credentials...")
	checking := modal.NewRows(controls).
		Row(progress, 1).
		Build()
	settings := modal.NewRows(controls).
		Row(fields, 7).
		Spacer().
		Row(d.message, 1).
		Actions(buttons)
	layout, pages := modal.NewPagedLayout(
		controls,
		" Settings ",
		72,
		modal.Page{
			Name: "checking", Rows: checking, Visible: true, Centered: true,
		},
		modal.Page{Name: "settings", Rows: settings},
	)
	d.pages = pages
	return layout
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

func credentialRow(
	controls components.Factory,
	label string,
	status tview.Primitive,
	primary tview.Primitive,
	clear tview.Primitive,
) tview.Primitive {
	labelView := controls.TextView()
	labelView.SetText(label)
	return controls.Grid().
		SetRows(1).
		SetColumns(settingsLabelWidth, 0, 2, 10, 1, 10).
		AddItem(labelView, 0, 0, 1, 1, 0, 0, false).
		AddItem(status, 0, 1, 1, 1, 0, 0, false).
		AddItem(primary, 0, 3, 1, 1, 0, 0, false).
		AddItem(clear, 0, 5, 1, 1, 0, 0, false)
}

func centeredButtons(
	controls components.Factory,
	first,
	second tview.Primitive,
) tview.Primitive {
	return controls.Flex(tview.FlexColumn).
		AddItem(nil, 0, 1, false).
		AddItem(first, 12, 0, false).
		AddItem(nil, 2, 0, false).
		AddItem(second, 12, 0, false).
		AddItem(nil, 0, 1, false)
}
