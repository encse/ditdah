package tui

import (
	"context"
	"errors"
	"strings"
	"testing"

	"morsemanual/internal/audio"
	"morsemanual/internal/callsign"
	domain "morsemanual/internal/settings"
	"morsemanual/internal/tui/components"
	"morsemanual/internal/tui/modal"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

func TestOpenShowsProgressUntilCredentialValidationFinishes(t *testing.T) {
	store := &recordingStore{loaded: domain.Settings{
		StationCallsign: "HA7NCS",
		QRZPassword:     "password",
	}}
	service := &recordingQRZService{}
	host := newTestHost()

	dialog := newDialog(
		t.Context(), host, store, service, store.loaded, nil, nil,
	)
	if len(dialog.Focusables()) != 0 {
		t.Fatal("settings controls are focusable while validation is running")
	}
	if page := dialog.pages.Active(); page != "checking" {
		t.Fatalf("front page = %q, want checking", page)
	}

	dialog.finishValidation(nil)
	if page := dialog.pages.Active(); page != "settings" {
		t.Fatalf("front page = %q, want settings", page)
	}
}

func TestOpenValidatesStoredQRZCredentials(t *testing.T) {
	store := &recordingStore{loaded: domain.Settings{
		StationCallsign: "HA7NCS",
		QRZPassword:     "wrong-password",
		QRZAPIKey:       "wrong-key",
	}}
	service := &recordingQRZService{
		loginErr:  errors.New("password incorrect"),
		apiKeyErr: errors.New("invalid key"),
	}
	host := newTestHost()

	Open(t.Context(), host, store, service, nil, nil)
	dialog := host.lastDialog().(*dialog)

	if service.callsign != "HA7NCS" || service.password != "wrong-password" {
		t.Fatalf("validated login = %q, %q", service.callsign, service.password)
	}
	if service.apiKey != "wrong-key" {
		t.Fatalf("validated API key = %q", service.apiKey)
	}
	if !strings.Contains(dialog.loginStatus.Text(), "Check failed") ||
		!strings.Contains(dialog.apiKeyStatus.Text(), "Check failed") {
		t.Fatalf(
			"statuses = %q, %q",
			dialog.loginStatus.Text(),
			dialog.apiKeyStatus.Text(),
		)
	}
}

func TestSettingsSelectsSavedMorseInput(t *testing.T) {
	store := &recordingStore{loaded: domain.Settings{
		MorseInputDeviceID: "usb",
	}}
	host := newTestHost()
	Open(
		t.Context(),
		host,
		store,
		&recordingQRZService{},
		recordingDeviceLister{devices: []audio.Device{
			{ID: "built-in", Name: "Built-in input", IsDefault: true},
			{ID: "usb", Name: "USB radio"},
		}},
		nil,
	)
	dialog := host.lastDialog().(*dialog)

	index, label := dialog.morseInput.CurrentOption()
	if index != 1 || label != "USB radio" {
		t.Fatalf("selected input = %d, %q, want saved USB device", index, label)
	}
}

func TestSettingsSavesDefaultMorseInputAndNotifiesApplication(t *testing.T) {
	values := domain.Settings{
		StationCallsign: "HA7NCS",
		QRZAPIKey:       "qrz-key",
	}
	store := &recordingStore{loaded: values}
	host := newTestHost()
	changes := 0
	Open(
		t.Context(),
		host,
		store,
		&recordingQRZService{},
		recordingDeviceLister{devices: []audio.Device{
			{ID: "first", Name: "Line input"},
			{ID: "default", Name: "USB radio", IsDefault: true},
		}},
		func() { changes++ },
	)
	dialog := host.lastDialog().(*dialog)

	dialog.submit()

	want := values
	want.MorseInputDeviceID = "default"
	if store.saved != want {
		t.Fatalf("saved settings = %#v, want %#v", store.saved, want)
	}
	if changes != 1 {
		t.Fatalf("settings change notifications = %d, want 1", changes)
	}
	if !host.lastHandle().closed {
		t.Fatal("successful Settings save did not close dialog")
	}
}

func TestSettingsNotifiesApplicationAfterChangedValuesWereStaged(t *testing.T) {
	values := domain.Settings{MorseInputDeviceID: "built-in"}
	store := &recordingStore{loaded: values}
	host := newTestHost()
	changes := 0
	dialog := newDialog(
		t.Context(),
		host,
		store,
		&recordingQRZService{},
		values,
		[]audio.Device{{ID: "usb", Name: "USB radio"}},
		func() { changes++ },
	)
	staged := values
	staged.StationCallsign = "HA7NCS"
	dialog.stage(staged)

	if err := dialog.save(staged); err != nil {
		t.Fatalf("save staged settings: %v", err)
	}
	if changes != 1 {
		t.Fatalf("settings change notifications = %d, want 1", changes)
	}
}

func TestSettingsShowsMorseInputEnumerationError(t *testing.T) {
	host := newTestHost()
	Open(
		t.Context(),
		host,
		&recordingStore{},
		&recordingQRZService{},
		recordingDeviceLister{err: errors.New("capture backend failed")},
		nil,
	)
	dialog := host.lastDialog().(*dialog)
	if !strings.Contains(dialog.message.Text(), "capture backend failed") {
		t.Fatalf("message = %q, want device enumeration error", dialog.message.Text())
	}
}

func TestSettingsButtonsReceiveMouseClicks(t *testing.T) {
	store := &recordingStore{}
	host := newTestHost()
	dialog := newDialog(
		t.Context(),
		host,
		store,
		&recordingQRZService{},
		domain.Settings{},
		nil,
		nil,
	)
	handle := &testHandle{}
	dialog.handle = handle
	dialog.finishValidation(nil)
	size := dialog.Size()
	dialog.Content().SetRect(0, 0, size.Width, size.Height)

	screen := tcell.NewSimulationScreen("UTF-8")
	if err := screen.Init(); err != nil {
		t.Fatalf("initialize screen: %v", err)
	}
	t.Cleanup(screen.Fini)
	screen.SetSize(size.Width, size.Height)
	dialog.Content().Draw(screen)

	x, y, width, _ := dialog.cancel.GetRect()
	consumed, _ := dialog.Content().MouseHandler()(
		tview.MouseLeftClick,
		tcell.NewEventMouse(x+width/2, y, tcell.Button1, 0),
		func(tview.Primitive) {},
	)
	if !consumed {
		t.Fatal("Cancel mouse click was not consumed")
	}
	if !handle.closed {
		t.Fatal("Cancel mouse click did not close settings")
	}

	handle = &testHandle{}
	dialog.handle = handle
	x, y, width, _ = dialog.ok.GetRect()
	consumed, _ = dialog.Content().MouseHandler()(
		tview.MouseLeftClick,
		tcell.NewEventMouse(x+width/2, y, tcell.Button1, 0),
		func(tview.Primitive) {},
	)
	if !consumed {
		t.Fatal("OK mouse click was not consumed")
	}
	if store.saveCalls != 1 || !handle.closed {
		t.Fatal("OK mouse click did not save and close settings")
	}
}

func TestLoginValidatesAndStagesPasswordUntilSettingsAreSaved(t *testing.T) {
	store := &recordingStore{loaded: domain.Settings{
		StationCallsign: "HA7NCS",
		QRZAPIKey:       "existing-key",
	}}
	service := &recordingQRZService{}
	host := newTestHost()
	Open(t.Context(), host, store, service, nil, nil)
	settings := host.lastDialog().(*dialog)

	settings.openLogin()
	login := host.lastDialog().(*loginDialog)
	login.password.SetValue("secret")
	login.submit()

	if service.callsign != "HA7NCS" || service.password != "secret" {
		t.Fatalf("validated login = %q, %q", service.callsign, service.password)
	}
	want := domain.Settings{
		StationCallsign: "HA7NCS",
		QRZPassword:     "secret",
		QRZAPIKey:       "existing-key",
	}
	if store.saveCalls != 0 {
		t.Fatalf("save calls before OK = %d, want 0", store.saveCalls)
	}
	if !host.lastHandle().closed {
		t.Fatal("successful login did not close its dialog")
	}
	if host.handles[0].closed {
		t.Fatal("successful login closed the settings dialog")
	}
	if settings.loginStatus.Text() != "Connected" {
		t.Fatalf("login status = %q", settings.loginStatus.Text())
	}

	settings.submit()
	if store.saved != want {
		t.Fatalf("saved settings = %#v, want %#v", store.saved, want)
	}
}

func TestFailedLoginRemainsOpenAndIsNotSaved(t *testing.T) {
	store := &recordingStore{loaded: domain.Settings{StationCallsign: "HA7NCS"}}
	service := &recordingQRZService{loginErr: errors.New("password incorrect")}
	host := newTestHost()
	Open(t.Context(), host, store, service, nil, nil)
	settings := host.lastDialog().(*dialog)

	settings.openLogin()
	login := host.lastDialog().(*loginDialog)
	login.password.SetValue("wrong")
	login.submit()

	if host.lastHandle().closed {
		t.Fatal("failed login closed its dialog")
	}
	if store.saveCalls != 0 {
		t.Fatalf("save calls = %d, want 0", store.saveCalls)
	}
	if !strings.Contains(login.message.Text(), "password incorrect") {
		t.Fatalf("login error = %q", login.message.Text())
	}
}

func TestAPIKeyUpdateValidatesAndStagesKeyUntilSettingsAreSaved(t *testing.T) {
	store := &recordingStore{loaded: domain.Settings{StationCallsign: "HA7NCS"}}
	service := &recordingQRZService{}
	host := newTestHost()
	Open(t.Context(), host, store, service, nil, nil)
	settings := host.lastDialog().(*dialog)

	settings.openAPIKey()
	editor := host.lastDialog().(*apiKeyDialog)
	editor.apiKey.SetValue("valid-key")
	editor.submit()

	if service.apiKey != "valid-key" {
		t.Fatalf("validated API key = %q", service.apiKey)
	}
	if store.saveCalls != 0 {
		t.Fatalf("save calls before OK = %d, want 0", store.saveCalls)
	}
	if !host.lastHandle().closed {
		t.Fatal("successful API key update did not close its dialog")
	}
	if settings.apiKeyStatus.Text() != "Set" {
		t.Fatalf("API key status = %q", settings.apiKeyStatus.Text())
	}

	settings.submit()
	if store.saved.QRZAPIKey != "valid-key" {
		t.Fatalf("saved API key = %q", store.saved.QRZAPIKey)
	}
}

func TestClearActionsStageEmptyCredentialsUntilSettingsAreSaved(t *testing.T) {
	store := &recordingStore{loaded: domain.Settings{
		StationCallsign: "HA7NCS",
		QRZPassword:     "password",
		QRZAPIKey:       "key",
	}}
	host := newTestHost()
	Open(t.Context(), host, store, &recordingQRZService{}, nil, nil)
	dialog := host.lastDialog().(*dialog)

	dialog.clearQRZLogin()
	dialog.clearQRZAPIKey()

	if store.saveCalls != 0 {
		t.Fatalf("save calls before OK = %d, want 0", store.saveCalls)
	}
	if dialog.loginStatus.Text() != "Not connected" ||
		dialog.apiKeyStatus.Text() != "Not set" {
		t.Fatalf(
			"statuses = %q, %q",
			dialog.loginStatus.Text(),
			dialog.apiKeyStatus.Text(),
		)
	}

	dialog.submit()
	if store.saved.QRZPassword != "" || store.saved.QRZAPIKey != "" {
		t.Fatalf("saved credentials = %#v", store.saved)
	}
}

func TestSubmitSavesCallsignAndClosesSettings(t *testing.T) {
	store := &recordingStore{}
	host := newTestHost()
	Open(t.Context(), host, store, &recordingQRZService{}, nil, nil)
	dialog := host.lastDialog().(*dialog)

	dialog.stationCallsign.SetValue("OE1ABC")
	dialog.submit()

	if store.saved.StationCallsign != "OE1ABC" {
		t.Fatalf("saved callsign = %q", store.saved.StationCallsign)
	}
	if !host.handles[0].closed {
		t.Fatal("successful save did not close settings dialog")
	}
}

func TestCancelDiscardsStagedCredentialChanges(t *testing.T) {
	store := &recordingStore{loaded: domain.Settings{
		StationCallsign: "HA7NCS",
		QRZPassword:     "password",
		QRZAPIKey:       "key",
	}}
	host := newTestHost()
	Open(t.Context(), host, store, &recordingQRZService{}, nil, nil)
	dialog := host.lastDialog().(*dialog)

	dialog.clearQRZLogin()
	dialog.clearQRZAPIKey()
	dialog.close()

	if store.saveCalls != 0 {
		t.Fatalf("save calls after Cancel = %d, want 0", store.saveCalls)
	}
	if !host.handles[0].closed {
		t.Fatal("Cancel did not close settings dialog")
	}
}

type recordingStore struct {
	loaded    domain.Settings
	saved     domain.Settings
	loadErr   error
	saveErr   error
	saveCalls int
}

func (s *recordingStore) Load(context.Context) (domain.Settings, error) {
	return s.loaded, s.loadErr
}

func (s *recordingStore) Save(
	_ context.Context,
	values domain.Settings,
) (domain.Settings, error) {
	s.saveCalls++
	if s.saveErr != nil {
		return domain.Settings{}, s.saveErr
	}
	s.saved = values
	s.loaded = values
	return values, nil
}

type recordingQRZService struct {
	callsign  string
	password  string
	apiKey    string
	loginErr  error
	apiKeyErr error
}

type recordingDeviceLister struct {
	devices []audio.Device
	err     error
}

func (l recordingDeviceLister) Devices() ([]audio.Device, error) {
	return l.devices, l.err
}

func (s *recordingQRZService) ValidateLogin(
	_ context.Context,
	callsign string,
	password string,
) error {
	s.callsign = callsign
	s.password = password
	return s.loginErr
}

func (s *recordingQRZService) ValidateAPIKey(
	_ context.Context,
	apiKey string,
) error {
	s.apiKey = apiKey
	return s.apiKeyErr
}

func (s *recordingQRZService) LookupCallsign(
	context.Context,
	string,
	string,
	string,
) (callsign.Record, error) {
	return callsign.Record{}, nil
}

type testHost struct {
	controls components.Factory
	dialogs  []modal.Dialog
	handles  []*testHandle
}

func newTestHost() *testHost {
	theme := components.Theme{
		Background:             tcell.ColorSilver,
		PrimaryText:            tcell.ColorWhite,
		SecondaryText:          tcell.ColorWhite,
		MutedText:              tcell.ColorGray,
		DangerText:             tcell.ColorRed,
		Accent:                 tcell.ColorWhite,
		Border:                 tcell.ColorWhite,
		LabelColor:             tcell.ColorWhite,
		FieldTextColor:         tcell.ColorBlack,
		FieldBackground:        tcell.ColorGray,
		ActiveFieldBackground:  tcell.ColorWhite,
		CursorColor:            tcell.ColorAqua,
		SelectionText:          tcell.ColorWhite,
		SelectionBackground:    tcell.ColorBlue,
		PopupBorder:            tcell.ColorWhite,
		ButtonText:             tcell.ColorWhite,
		ButtonBackground:       tcell.ColorBlue,
		ActiveButtonText:       tcell.ColorBlack,
		ActiveButtonBackground: tcell.ColorAqua,
	}
	return &testHost{
		controls: components.New(components.Dependencies{Theme: theme}),
	}
}

func (h *testHost) SetFocus(tview.Primitive) {}

func (h *testHost) Refresh() {}

func (h *testHost) Update(update func()) {
	if update != nil {
		update()
	}
}

func (h *testHost) Components() components.Factory { return h.controls }

func (h *testHost) OpenModal(dialog modal.Dialog) modal.Handle {
	handle := &testHandle{}
	h.dialogs = append(h.dialogs, dialog)
	h.handles = append(h.handles, handle)
	return handle
}

func (h *testHost) lastDialog() modal.Dialog {
	return h.dialogs[len(h.dialogs)-1]
}

func (h *testHost) lastHandle() *testHandle {
	return h.handles[len(h.handles)-1]
}

type testHandle struct {
	closed bool
}

func (h *testHandle) Close() { h.closed = true }
