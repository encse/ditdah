// Package tui implements the settings terminal dialogs.
package tui

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync/atomic"

	"ditdah/internal/audio"
	"ditdah/internal/qrz"
	"ditdah/internal/radio"
	domain "ditdah/internal/settings"
	"ditdah/internal/syncutil"
	ui "ditdah/internal/tui"
	"ditdah/internal/tui/components"
	"ditdah/internal/tui/keybinding"
	"ditdah/internal/tui/modal"

	"github.com/rivo/tview"
	"golang.org/x/sync/errgroup"
)

const settingsLabelWidth = 20

type dialog struct {
	modal.Layout
	host       ui.PageHost
	store      domain.Store
	qrz        qrz.Service
	values     domain.Settings
	handle     modal.Handle
	devices    []audio.Device
	loadErr    error
	devicesErr error
	inputs     audio.DeviceLister
	radio      radio.Service

	stationCallsign  components.InputField
	morseInput       components.SelectField
	loginStatus      components.TextView
	apiKeyStatus     components.TextView
	radioInfo        components.TextView
	radioStatus      components.TextView
	message          components.TextView
	login            components.Button
	clearLogin       components.Button
	updateAPIKey     components.Button
	clearAPIKey      components.Button
	configureRadio   components.Button
	ok               components.Button
	cancel           components.Button
	focusables       []tview.Primitive
	loginChecks      syncutil.Mailbox[loginValidationRequest]
	loginGeneration  atomic.Uint64
	apiKeyGeneration atomic.Uint64
	radioGeneration  atomic.Uint64
}

type loginValidationRequest struct {
	generation uint64
	callsign   string
	password   string
}

type applicationHost interface {
	ui.PageHost
	OpenModalForCurrentLayer(dialog modal.Dialog) modal.Handle
}

// Open displays the settings editor. Loading and credential validation belong
// to the dialog's Run lifecycle.
func Open(
	host applicationHost,
	store domain.Store,
	qrzService qrz.Service,
	inputs audio.DeviceLister,
	radios ...radio.Service,
) {
	dialog := newDialog(host, store, qrzService, inputs, radios...)
	dialog.handle = host.OpenModalForCurrentLayer(dialog)
}

func newDialog(
	host ui.PageHost,
	store domain.Store,
	qrzService qrz.Service,
	inputs audio.DeviceLister,
	radios ...radio.Service,
) *dialog {
	controls := host.Components().Modal()
	dialog := &dialog{
		host:   host,
		store:  store,
		qrz:    qrzService,
		inputs: inputs,
	}
	if len(radios) > 0 {
		dialog.radio = radios[0]
	}
	dialog.stationCallsign = dialog.input(
		controls,
		"My callsign",
		"",
	)
	dialog.loginGeneration.Store(1)
	dialog.apiKeyGeneration.Store(1)
	dialog.loginChecks = syncutil.NewMailbox(loginValidationRequest{
		generation: 1,
	})
	dialog.stationCallsign.SetChangedFunc(dialog.callsignChanged)
	dialog.morseInput = controls.SelectField(
		"Audio input",
		nil,
		-1,
		settingsLabelWidth,
		0,
	)
	dialog.loginStatus = controls.TextView()
	dialog.apiKeyStatus = controls.TextView()
	dialog.radioInfo = controls.TextView()
	dialog.radioStatus = controls.TextView()
	dialog.message = controls.TextView()
	dialog.message.SetStyle(components.TextViewDanger)
	dialog.message.SetTextAlign(tview.AlignCenter)
	dialog.login = controls.Button("Login")
	dialog.clearLogin = controls.Button("Clear")
	dialog.updateAPIKey = controls.Button("Update")
	dialog.clearAPIKey = controls.Button("Clear")
	dialog.configureRadio = controls.Button("Configure")
	dialog.ok = controls.Button("OK")
	dialog.cancel = controls.Button("Cancel")
	dialog.login.SetSelectedFunc(dialog.openLogin)
	dialog.clearLogin.SetSelectedFunc(dialog.clearQRZLogin)
	dialog.updateAPIKey.SetSelectedFunc(dialog.openAPIKey)
	dialog.clearAPIKey.SetSelectedFunc(dialog.clearQRZAPIKey)
	dialog.configureRadio.SetSelectedFunc(dialog.openRadio)
	dialog.ok.SetSelectedFunc(dialog.submit)
	dialog.cancel.SetSelectedFunc(dialog.close)
	dialog.focusables = []tview.Primitive{
		dialog.stationCallsign,
		dialog.morseInput,
		dialog.login,
		dialog.clearLogin,
		dialog.updateAPIKey,
		dialog.clearAPIKey,
		dialog.configureRadio,
		dialog.ok,
		dialog.cancel,
	}
	dialog.Layout = dialog.layout(controls)
	dialog.showInitialCredentialStatuses()
	return dialog
}

func (d *dialog) Focusables() []tview.Primitive {
	return d.focusables
}

func (d *dialog) KeyBindings() []keybinding.Binding {
	return nil
}

func (d *dialog) Run(ctx context.Context) {
	apiKey, loaded := d.load(ctx)
	if !loaded {
		<-ctx.Done()
		return
	}

	group, runCtx := errgroup.WithContext(ctx)
	group.Go(func() error { return d.runLoginChecks(runCtx) })
	group.Go(func() error {
		d.validateStoredAPIKey(runCtx, apiKey)
		return nil
	})
	group.Go(func() error {
		d.validateStoredRadio(runCtx)
		return nil
	})
	_ = group.Wait()
}

func (d *dialog) load(ctx context.Context) (string, bool) {
	values, loadErr := d.store.Load(ctx)
	var devices []audio.Device
	var devicesErr error
	if d.inputs == nil {
		devicesErr = errors.New("audio input is unavailable")
	} else {
		devices, devicesErr = d.inputs.Devices()
	}
	if ctx.Err() != nil {
		return "", false
	}
	d.host.Update(d, func() {
		d.values = values
		d.devices = append(d.devices[:0], devices...)
		d.loadErr = loadErr
		d.devicesErr = devicesErr
		d.message.SetText("")
		d.stationCallsign.SetValue(values.StationCallsign)
		d.morseInput.SetOptions(
			deviceOptions(devices),
			selectedMorseInput(values.MorseInputDeviceID, devices),
		)
		d.showInitialCredentialStatuses()
		d.showRadioInfo()
		d.showInitialRadioStatus()
		if loadErr != nil {
			d.showError(fmt.Errorf("load settings: %w", loadErr))
		}
		d.showInitialDeviceError()
	})
	return values.QRZAPIKey, loadErr == nil
}

func deviceOptions(devices []audio.Device) []string {
	options := make([]string, len(devices))
	for index, device := range devices {
		options[index] = device.Name
		if device.IsDefault {
			options[index] += " (default)"
		}
	}
	return options
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

func (d *dialog) runLoginChecks(ctx context.Context) error {
	for {
		request, err := d.loginChecks.Receive(ctx)
		if err != nil {
			return nil
		}
		var validationErr error
		switch {
		case request.password == "":
		case request.callsign == "":
			validationErr = errors.New("callsign is required")
		default:
			validationErr = d.qrz.ValidateLogin(
				ctx,
				request.callsign,
				request.password,
			)
		}
		if ctx.Err() != nil {
			return nil
		}
		d.host.Update(d, func() {
			if d.loginGeneration.Load() == request.generation {
				d.showLoginStatusFor(request.password, validationErr)
			}
		})
	}
}

func (d *dialog) validateStoredAPIKey(ctx context.Context, apiKey string) {
	generation := d.apiKeyGeneration.Load()
	if apiKey == "" {
		return
	}
	validationErr := d.qrz.ValidateAPIKey(ctx, apiKey)
	if ctx.Err() != nil {
		return
	}
	d.host.Update(d, func() {
		if d.apiKeyGeneration.Load() == generation {
			d.showAPIKeyStatus(validationErr)
		}
	})
}

func (d *dialog) showInitialDeviceError() {
	switch {
	case d.devicesErr != nil:
		d.showError(errors.Join(d.loadErr, d.devicesErr))
	case len(d.devices) == 0:
		d.showError(errors.New("no audio input devices found"))
	case d.values.MorseInputDeviceID != "":
		index, _ := d.morseInput.CurrentOption()
		if index < 0 {
			d.showError(errors.New("the selected audio input is unavailable"))
		}
	}
}

func (d *dialog) submit() {
	values := d.currentValues()
	d.host.Background(d, func(ctx context.Context) {
		saved, err := d.store.Save(ctx, values)
		if ctx.Err() == nil {
			d.host.Update(d, func() { d.finishSave(saved, err) })
		}
	})
}

func (d *dialog) openLogin() {
	callsign := strings.TrimSpace(d.stationCallsign.Value())
	if callsign == "" {
		d.showError(fmt.Errorf("callsign is required before QRZ.com login"))
		return
	}
	var login *loginDialog
	login = newLoginDialog(
		d.host.Components(),
		callsign,
		func(password string) {
			d.host.Background(login, func(ctx context.Context) {
				err := d.qrz.ValidateLogin(ctx, callsign, password)
				if ctx.Err() != nil {
					return
				}
				d.host.Update(login, func() {
					if err == nil {
						values := d.currentValues()
						values.StationCallsign = callsign
						values.QRZPassword = password
						d.stage(values)
						d.loginGeneration.Add(1)
						d.stationCallsign.SetValue(callsign)
						d.showLoginStatus(nil)
					}
					login.finish(err)
				})
			})
		},
	)
	login.handle = d.host.OpenModal(d, login)
}

func (d *dialog) openAPIKey() {
	var editor *apiKeyDialog
	editor = newAPIKeyDialog(
		d.host.Components(),
		func(apiKey string) {
			d.host.Background(editor, func(ctx context.Context) {
				err := d.qrz.ValidateAPIKey(ctx, apiKey)
				if ctx.Err() != nil {
					return
				}
				d.host.Update(editor, func() {
					if err == nil {
						values := d.currentValues()
						values.QRZAPIKey = apiKey
						d.stage(values)
						d.apiKeyGeneration.Add(1)
						d.showAPIKeyStatus(nil)
					}
					editor.finish(err)
				})
			})
		},
	)
	editor.handle = d.host.OpenModal(d, editor)
}

func (d *dialog) openRadio() {
	if d.radio == nil {
		d.showError(errors.New("Hamlib is unavailable"))
		return
	}
	values := d.currentValues()
	var editor *radioDialog
	editor = newRadioDialog(
		d.host,
		d.host.Components(),
		d.radio,
		radio.Config{
			ModelID:   values.RadioModelID,
			ModelName: values.RadioModelName,
			Port:      values.RadioSerialPort,
			BaudRate:  values.RadioBaudRate,
		},
		func(config radio.Config, frequency uint64) {
			d.radioGeneration.Add(1)
			values := d.currentValues()
			values.RadioModelID = config.ModelID
			values.RadioModelName = config.ModelName
			values.RadioSerialPort = config.Port
			values.RadioBaudRate = config.BaudRate
			d.stage(values)
			d.showRadioInfo()
			d.showRadioConnected(frequency)
		},
	)
	editor.handle = d.host.OpenModal(d, editor)
}

func (d *dialog) clearQRZLogin() {
	values := d.currentValues()
	values.QRZPassword = ""
	d.stage(values)
	d.loginGeneration.Add(1)
	d.showLoginStatus(nil)
}

func (d *dialog) clearQRZAPIKey() {
	values := d.currentValues()
	values.QRZAPIKey = ""
	d.stage(values)
	d.apiKeyGeneration.Add(1)
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

func (d *dialog) finishSave(saved domain.Settings, err error) {
	if err != nil {
		d.showError(err)
		return
	}
	d.values = saved
	d.message.SetText("")
	d.close()
}

func (d *dialog) stage(values domain.Settings) {
	d.values = values
	d.message.SetText("")
}

func (d *dialog) callsignChanged(value string) {
	password := d.values.QRZPassword
	generation := d.loginGeneration.Add(1)
	if password == "" {
		d.showLoginStatusFor("", nil)
	} else {
		d.showLoginChecking()
	}
	d.loginChecks.Send(loginValidationRequest{
		generation: generation,
		callsign:   strings.TrimSpace(value),
		password:   password,
	})
}

func (d *dialog) showInitialCredentialStatuses() {
	if d.values.QRZPassword == "" {
		d.showLoginStatus(nil)
	} else {
		d.showLoginChecking()
	}
	if d.values.QRZAPIKey == "" {
		d.showAPIKeyStatus(nil)
	} else {
		d.apiKeyStatus.SetStyle(components.TextViewMuted)
		d.apiKeyStatus.SetText("Checking...")
	}
}

func (d *dialog) showLoginChecking() {
	d.loginStatus.SetStyle(components.TextViewMuted)
	d.loginStatus.SetText("Checking...")
}

func (d *dialog) showLoginStatus(validationErr error) {
	d.showLoginStatusFor(d.values.QRZPassword, validationErr)
}

func (d *dialog) showLoginStatusFor(
	password string,
	validationErr error,
) {
	switch {
	case password == "":
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

func (d *dialog) validateStoredRadio(ctx context.Context) {
	config := d.radioConfig()
	generation := d.radioGeneration.Load()
	if config.ModelID == 0 {
		return
	}
	if d.radio == nil {
		d.host.Update(d, func() {
			d.showRadioFailure(errors.New("Hamlib is unavailable"))
		})
		return
	}
	frequency, err := d.radio.Check(ctx, config)
	if ctx.Err() != nil {
		return
	}
	d.host.Update(d, func() {
		if generation != d.radioGeneration.Load() {
			return
		}
		if err != nil {
			d.showRadioFailure(err)
			return
		}
		d.showRadioConnected(frequency)
	})
}

func (d *dialog) radioConfig() radio.Config {
	return radio.Config{
		ModelID:   d.values.RadioModelID,
		ModelName: d.values.RadioModelName,
		Port:      d.values.RadioSerialPort,
		BaudRate:  d.values.RadioBaudRate,
	}
}

func (d *dialog) showRadioInfo() {
	config := d.radioConfig()
	if config.ModelID == 0 {
		d.radioInfo.SetStyle(components.TextViewMuted)
		d.radioInfo.SetText("Not configured")
		return
	}
	d.radioInfo.SetStyle(components.TextViewPrimary)
	d.radioInfo.SetText(fmt.Sprintf(
		"%s — %s @ %d",
		config.ModelName,
		config.Port,
		config.BaudRate,
	))
}

func (d *dialog) showInitialRadioStatus() {
	if d.values.RadioModelID == 0 {
		d.radioStatus.SetStyle(components.TextViewMuted)
		d.radioStatus.SetText("")
		return
	}
	d.radioStatus.SetStyle(components.TextViewMuted)
	d.radioStatus.SetText("Checking...")
}

func (d *dialog) showRadioConnected(frequency uint64) {
	d.radioStatus.SetStyle(components.TextViewAccent)
	d.radioStatus.SetText(formatFrequency(frequency))
}

func (d *dialog) showRadioFailure(err error) {
	d.radioStatus.SetStyle(components.TextViewDanger)
	d.radioStatus.SetText("Check failed: " + err.Error())
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
	radioRow := settingActionRow(
		controls,
		"Radio",
		d.radioInfo,
		d.configureRadio,
	)
	radioStatusRow := settingStatusRow(controls, d.radioStatus)
	fields := controls.Grid().
		SetRows(1, 1, 1, 1, 1, 1, 1, 1, 1, 1).
		SetColumns(0).
		AddItem(d.stationCallsign, 0, 0, 1, 1, 0, 0, false).
		AddItem(d.morseInput, 2, 0, 1, 1, 0, 0, false).
		AddItem(loginRow, 4, 0, 1, 1, 0, 0, false).
		AddItem(apiKeyRow, 6, 0, 1, 1, 0, 0, false).
		AddItem(radioRow, 8, 0, 1, 1, 0, 0, false).
		AddItem(radioStatusRow, 9, 0, 1, 1, 0, 0, false)
	buttons := centeredButtons(controls, d.ok, d.cancel)
	return modal.NewLayout(controls, " Settings ", 72).
		Row(fields, 10).
		Spacer().
		Row(d.message, 1).
		Actions(buttons)
}

func settingActionRow(
	controls components.Factory,
	label string,
	info tview.Primitive,
	action tview.Primitive,
) tview.Primitive {
	labelView := controls.TextView()
	labelView.SetText(label)
	return controls.Grid().
		SetRows(1).
		SetColumns(settingsLabelWidth, 0, 2, 12).
		AddItem(labelView, 0, 0, 1, 1, 0, 0, false).
		AddItem(info, 0, 1, 1, 1, 0, 0, false).
		AddItem(action, 0, 3, 1, 1, 0, 0, false)
}

func settingStatusRow(
	controls components.Factory,
	status tview.Primitive,
) tview.Primitive {
	return controls.Grid().
		SetRows(1).
		SetColumns(settingsLabelWidth, 0).
		AddItem(status, 0, 1, 1, 1, 0, 0, false)
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
