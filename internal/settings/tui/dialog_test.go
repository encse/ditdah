package tui

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"ditdah/internal/audio"
	"ditdah/internal/callsign"
	domain "ditdah/internal/settings"
	"ditdah/internal/syncutil"
	ui "ditdah/internal/tui"
	"ditdah/internal/tui/components"
	"ditdah/internal/tui/modal"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

func TestSettingsIsUsableWhileCredentialValidationRuns(t *testing.T) {
	store := &recordingStore{loaded: domain.Settings{
		StationCallsign: "HA7NCS",
		QRZPassword:     "password",
	}}
	service := newControlledQRZService()
	host := newTestHost()
	host.updated = make(chan struct{}, 1)

	dialog := newDialog(host, store, service, nil, nil)
	if len(dialog.Focusables()) == 0 {
		t.Fatal("settings controls are not immediately focusable")
	}

	cancel, done := runSettingsDialog(t, dialog)
	defer stopSettingsDialog(t, cancel, done)
	<-host.updated
	if dialog.loginStatus.Text() != "Checking..." {
		t.Fatalf("login status = %q, want Checking...", dialog.loginStatus.Text())
	}

	request := <-service.loginRequests
	if request.callsign != "HA7NCS" || request.password != "password" {
		t.Fatalf("login request = %#v", request)
	}
}

func TestOpenValidatesStoredQRZCredentials(t *testing.T) {
	store := &recordingStore{loaded: domain.Settings{
		StationCallsign: "HA7NCS",
		QRZPassword:     "wrong-password",
		QRZAPIKey:       "wrong-key",
	}}
	service := newControlledQRZService()
	host := newTestHost()
	host.updated = make(chan struct{}, 3)

	Open(host, store, service, nil, nil)
	dialog := host.lastDialog().(*dialog)
	cancel, done := runSettingsDialog(t, dialog)
	defer stopSettingsDialog(t, cancel, done)
	<-host.updated
	loginRequest := <-service.loginRequests
	apiKey := <-service.apiKeyRequests
	service.loginResults <- errors.New("password incorrect")
	service.apiKeyResults <- errors.New("invalid key")
	<-host.updated
	<-host.updated

	if loginRequest.callsign != "HA7NCS" ||
		loginRequest.password != "wrong-password" {
		t.Fatalf("validated login = %#v", loginRequest)
	}
	if apiKey != "wrong-key" {
		t.Fatalf("validated API key = %q", apiKey)
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

func TestSettingsRevalidatesQRZLoginWhenCallsignChanges(t *testing.T) {
	values := domain.Settings{
		StationCallsign: "HA7NCS",
		QRZPassword:     "password",
	}
	service := newControlledQRZService()
	host := newTestHost()
	host.updated = make(chan struct{}, 3)
	dialog := newDialog(
		host, &recordingStore{loaded: values}, service, nil, nil,
	)
	cancel, done := runSettingsDialog(t, dialog)
	defer stopSettingsDialog(t, cancel, done)
	<-host.updated

	initial := <-service.loginRequests
	if initial.callsign != "HA7NCS" {
		t.Fatalf("initial callsign = %q", initial.callsign)
	}

	dialog.stationCallsign.SetValue("OE1ABC")
	if dialog.loginStatus.Text() != "Checking..." {
		t.Fatalf("changed login status = %q", dialog.loginStatus.Text())
	}
	service.loginResults <- errors.New("old callsign failed")
	<-host.updated
	if dialog.loginStatus.Text() != "Checking..." {
		t.Fatalf("stale result changed login status to %q", dialog.loginStatus.Text())
	}
	changed := <-service.loginRequests
	if changed.callsign != "OE1ABC" || changed.password != "password" {
		t.Fatalf("changed login request = %#v", changed)
	}
	service.loginResults <- nil
	<-host.updated
	if dialog.loginStatus.Text() != "Connected" {
		t.Fatalf("validated login status = %q", dialog.loginStatus.Text())
	}
}

func TestSettingsSelectsSavedMorseInput(t *testing.T) {
	store := &recordingStore{loaded: domain.Settings{
		MorseInputDeviceID: "usb",
	}}
	host := newTestHost()
	Open(
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
	if store.loadCalls != 0 {
		t.Fatal("Open loaded settings before the dialog Run lifecycle started")
	}
	loadSettingsDialog(t, host, dialog)
	if store.loadCalls != 1 {
		t.Fatalf("settings loads = %d, want one Run-owned load", store.loadCalls)
	}

	index, label := dialog.morseInput.CurrentOption()
	if index != 1 || label != "USB radio" {
		t.Fatalf("selected input = %d, %q, want saved USB device", index, label)
	}
}

func TestSettingsSavesDefaultMorseInput(t *testing.T) {
	values := domain.Settings{
		StationCallsign: "HA7NCS",
		QRZAPIKey:       "qrz-key",
	}
	store := &recordingStore{loaded: values}
	host := newTestHost()
	Open(
		host,
		store,
		&recordingQRZService{},
		recordingDeviceLister{devices: []audio.Device{
			{ID: "first", Name: "Line input"},
			{ID: "default", Name: "USB radio", IsDefault: true},
		}},
		nil,
	)
	dialog := host.lastDialog().(*dialog)
	loadSettingsDialog(t, host, dialog)

	dialog.submit()

	want := values
	want.MorseInputDeviceID = "default"
	if store.saved != want {
		t.Fatalf("saved settings = %#v, want %#v", store.saved, want)
	}
	if !host.lastHandle().closed {
		t.Fatal("successful Settings save did not close dialog")
	}
}

func TestSettingsShowsMorseInputEnumerationError(t *testing.T) {
	host := newTestHost()
	host.updated = make(chan struct{}, 1)
	Open(
		host,
		&recordingStore{},
		&recordingQRZService{},
		recordingDeviceLister{err: errors.New("capture backend failed")},
		nil,
	)
	dialog := host.lastDialog().(*dialog)
	cancel, done := runSettingsDialog(t, dialog)
	defer stopSettingsDialog(t, cancel, done)
	<-host.updated
	if !strings.Contains(dialog.message.Text(), "capture backend failed") {
		t.Fatalf("message = %q, want device enumeration error", dialog.message.Text())
	}
}

func TestSettingsButtonsReceiveMouseClicks(t *testing.T) {
	store := &recordingStore{}
	host := newTestHost()
	dialog := newDialog(
		host,
		store,
		&recordingQRZService{},
		nil,
		nil,
	)
	handle := &testHandle{}
	dialog.handle = handle
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
	Open(host, store, service, nil, nil)
	settings := host.lastDialog().(*dialog)
	loadSettingsDialog(t, host, settings)

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
	Open(host, store, service, nil, nil)
	settings := host.lastDialog().(*dialog)
	loadSettingsDialog(t, host, settings)

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
	Open(host, store, service, nil, nil)
	settings := host.lastDialog().(*dialog)
	loadSettingsDialog(t, host, settings)

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
	Open(host, store, &recordingQRZService{}, nil, nil)
	dialog := host.lastDialog().(*dialog)
	loadSettingsDialog(t, host, dialog)

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
	Open(host, store, &recordingQRZService{}, nil, nil)
	dialog := host.lastDialog().(*dialog)
	loadSettingsDialog(t, host, dialog)

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
	Open(host, store, &recordingQRZService{}, nil, nil)
	dialog := host.lastDialog().(*dialog)
	loadSettingsDialog(t, host, dialog)

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

func runSettingsDialog(
	t *testing.T,
	dialog *dialog,
) (context.CancelFunc, <-chan struct{}) {
	t.Helper()
	ctx, cancel := context.WithCancel(t.Context())
	done := make(chan struct{})
	go func() {
		dialog.Run(ctx)
		close(done)
	}()
	return cancel, done
}

func loadSettingsDialog(
	t *testing.T,
	host *testHost,
	dialog *dialog,
) {
	t.Helper()
	if host.updated == nil {
		host.updated = make(chan struct{}, 8)
	}
	cancel, done := runSettingsDialog(t, dialog)
	select {
	case <-host.updated:
	case <-time.After(time.Second):
		cancel()
		t.Fatal("settings were not loaded by Run")
	}
	stopSettingsDialog(t, cancel, done)
}

func stopSettingsDialog(
	t *testing.T,
	cancel context.CancelFunc,
	done <-chan struct{},
) {
	t.Helper()
	cancel()
	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("settings dialog Run did not stop")
	}
}

type recordingStore struct {
	loaded    domain.Settings
	saved     domain.Settings
	loadErr   error
	saveErr   error
	loadCalls int
	saveCalls int
}

func (s *recordingStore) Load(context.Context) (domain.Settings, error) {
	s.loadCalls++
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

func (s *recordingStore) Subscribe() syncutil.Subscription {
	return syncutil.NewBroadcaster().Subscribe()
}

type recordingQRZService struct {
	callsign  string
	password  string
	apiKey    string
	loginErr  error
	apiKeyErr error
}

type loginValidation struct {
	callsign string
	password string
}

type controlledQRZService struct {
	loginRequests  chan loginValidation
	loginResults   chan error
	apiKeyRequests chan string
	apiKeyResults  chan error
}

func newControlledQRZService() *controlledQRZService {
	return &controlledQRZService{
		loginRequests:  make(chan loginValidation, 1),
		loginResults:   make(chan error, 1),
		apiKeyRequests: make(chan string, 1),
		apiKeyResults:  make(chan error, 1),
	}
}

func (s *controlledQRZService) ValidateLogin(
	ctx context.Context,
	callsign string,
	password string,
) error {
	select {
	case s.loginRequests <- loginValidation{callsign, password}:
	case <-ctx.Done():
		return ctx.Err()
	}
	select {
	case result := <-s.loginResults:
		return result
	case <-ctx.Done():
		return ctx.Err()
	}
}

func (s *controlledQRZService) ValidateAPIKey(
	ctx context.Context,
	apiKey string,
) error {
	select {
	case s.apiKeyRequests <- apiKey:
	case <-ctx.Done():
		return ctx.Err()
	}
	select {
	case result := <-s.apiKeyResults:
		return result
	case <-ctx.Done():
		return ctx.Err()
	}
}

func (s *controlledQRZService) LookupCallsign(
	context.Context,
	string,
	string,
	string,
) (callsign.Record, error) {
	return callsign.Record{}, nil
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
	updated  chan struct{}
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

func (h *testHost) Update(_ ui.Owner, update func()) bool {
	if update != nil {
		update()
	}
	if h.updated != nil {
		h.updated <- struct{}{}
	}
	return true
}

func (h *testHost) Components() components.Factory { return h.controls }

func (h *testHost) OpenModal(
	_ ui.Owner,
	dialog modal.Dialog,
) modal.Handle {
	handle := &testHandle{}
	h.dialogs = append(h.dialogs, dialog)
	h.handles = append(h.handles, handle)
	return handle
}

func (h *testHost) OpenModalForCurrentLayer(
	dialog modal.Dialog,
) modal.Handle {
	return h.OpenModal(nil, dialog)
}

func (h *testHost) Background(
	_ ui.Owner,
	work ui.BackgroundWork,
) bool {
	work(context.Background())
	return true
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
