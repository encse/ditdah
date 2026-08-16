// Package tui implements the settings terminal dialogs.
package tui

import (
	"context"
	"fmt"
	"strings"

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
	ctx     context.Context
	host    ui.PageHost
	store   domain.Store
	qrz     qrz.Service
	content tview.Primitive
	values  domain.Settings
	handle  modal.Handle

	stationCallsign components.InputField
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
) {
	values, loadErr := store.Load(ctx)
	dialog := newDialog(ctx, host, store, qrzService, values)
	if loadErr != nil {
		dialog.showError(fmt.Errorf("load settings: %w", loadErr))
	} else {
		dialog.validateStoredCredentials()
	}
	dialog.handle = host.OpenModal(dialog)
}

func newDialog(
	ctx context.Context,
	host ui.PageHost,
	store domain.Store,
	qrzService qrz.Service,
	values domain.Settings,
) *dialog {
	controls := host.Components().Modal()
	dialog := &dialog{
		ctx:    ctx,
		host:   host,
		store:  store,
		qrz:    qrzService,
		values: values,
	}
	dialog.stationCallsign = dialog.input(
		controls,
		"My callsign",
		values.StationCallsign,
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
		dialog.login,
		dialog.clearLogin,
		dialog.updateAPIKey,
		dialog.clearAPIKey,
		dialog.ok,
		dialog.cancel,
	}
	dialog.content = dialog.layout(controls)
	dialog.showLoginStatus(nil)
	dialog.showAPIKeyStatus(nil)
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
	return modal.Size{Width: 72, Height: 12}
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

func (d *dialog) validateStoredCredentials() {
	if d.values.QRZPassword != "" {
		d.showLoginStatus(d.qrz.ValidateLogin(
			d.ctx,
			d.values.StationCallsign,
			d.values.QRZPassword,
		))
	}
	if d.values.QRZAPIKey != "" {
		d.showAPIKeyStatus(d.qrz.ValidateAPIKey(d.ctx, d.values.QRZAPIKey))
	}
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
	return values
}

func (d *dialog) save(values domain.Settings) error {
	saved, err := d.store.Save(d.ctx, values)
	if err != nil {
		return err
	}
	d.values = saved
	d.message.SetText("")
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

func (d *dialog) layout(controls components.Factory) tview.Primitive {
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
	fields := tview.NewGrid().
		SetRows(1, 1, 1, 1, 1).
		SetColumns(0).
		AddItem(d.stationCallsign, 0, 0, 1, 1, 0, 0, false).
		AddItem(loginRow, 2, 0, 1, 1, 0, 0, false).
		AddItem(apiKeyRow, 4, 0, 1, 1, 0, 0, false)
	buttons := centeredButtons(d.ok, d.cancel)
	body := tview.NewFlex().
		SetDirection(tview.FlexRow).
		AddItem(fields, 5, 0, false).
		AddItem(nil, 1, 0, false).
		AddItem(d.message, 1, 0, false).
		AddItem(buttons, 1, 0, false)
	padded := pad(body, 2, 3)
	surface := controls.TextView()
	surface.SetBorder(" Settings ")
	return tview.NewPages().
		AddPage("surface", surface, true, true).
		AddPage("content", padded, true, true)
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
	return tview.NewGrid().
		SetRows(1).
		SetColumns(settingsLabelWidth, 0, 2, 10, 1, 10).
		AddItem(labelView, 0, 0, 1, 1, 0, 0, false).
		AddItem(status, 0, 1, 1, 1, 0, 0, false).
		AddItem(primary, 0, 3, 1, 1, 0, 0, false).
		AddItem(clear, 0, 5, 1, 1, 0, 0, false)
}

func centeredButtons(first, second tview.Primitive) tview.Primitive {
	return tview.NewFlex().
		AddItem(nil, 0, 1, false).
		AddItem(first, 12, 0, false).
		AddItem(nil, 2, 0, false).
		AddItem(second, 12, 0, false).
		AddItem(nil, 0, 1, false)
}

func pad(content tview.Primitive, vertical, horizontal int) tview.Primitive {
	return tview.NewFlex().
		SetDirection(tview.FlexRow).
		AddItem(nil, vertical, 0, false).
		AddItem(
			tview.NewFlex().
				AddItem(nil, horizontal, 0, false).
				AddItem(content, 0, 1, false).
				AddItem(nil, horizontal, 0, false),
			0,
			1,
			false,
		).
		AddItem(nil, vertical, 0, false)
}
