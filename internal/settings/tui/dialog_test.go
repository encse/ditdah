package tui

import (
	"context"
	"errors"
	"strings"
	"testing"

	domain "morsemanual/internal/settings"
	"morsemanual/internal/tui/components"
	"morsemanual/internal/tui/modal"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

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

	Open(t.Context(), host, store, service)
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

func TestLoginValidatesAndStagesPasswordUntilSettingsAreSaved(t *testing.T) {
	store := &recordingStore{loaded: domain.Settings{
		StationCallsign: "HA7NCS",
		QRZAPIKey:       "existing-key",
	}}
	service := &recordingQRZService{}
	host := newTestHost()
	Open(t.Context(), host, store, service)
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
	Open(t.Context(), host, store, service)
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
	Open(t.Context(), host, store, service)
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
	Open(t.Context(), host, store, &recordingQRZService{})
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
	Open(t.Context(), host, store, &recordingQRZService{})
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
	Open(t.Context(), host, store, &recordingQRZService{})
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
