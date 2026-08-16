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

func TestOpenLoadsAndSavesSettings(t *testing.T) {
	store := &recordingStore{loaded: domain.Settings{
		StationCallsign: "HA7NCS",
		QRZPassword:     "old-password",
		QRZAPIKey:       "old-key",
	}}
	host := newTestHost()

	Open(t.Context(), host, store)
	dialog, ok := host.dialog.(*dialog)
	if !ok {
		t.Fatalf("opened dialog = %T, want *dialog", host.dialog)
	}
	if dialog.stationCallsign.Value() != "HA7NCS" ||
		dialog.qrzPassword.Value() != "old-password" ||
		dialog.qrzAPIKey.Value() != "old-key" {
		t.Fatalf("loaded fields = %#v", dialog.values)
	}

	dialog.stationCallsign.SetValue("OE1ABC")
	dialog.qrzPassword.SetValue("new-password")
	dialog.qrzAPIKey.SetValue("new-key")
	dialog.submit()

	want := domain.Settings{
		StationCallsign: "OE1ABC",
		QRZPassword:     "new-password",
		QRZAPIKey:       "new-key",
	}
	if store.saved != want {
		t.Fatalf("saved settings = %#v, want %#v", store.saved, want)
	}
	if !host.handle.closed {
		t.Fatal("successful save did not close settings dialog")
	}
}

func TestSaveErrorKeepsSettingsDialogOpen(t *testing.T) {
	store := &recordingStore{saveErr: errors.New("disk failed")}
	host := newTestHost()
	Open(t.Context(), host, store)
	dialog := host.dialog.(*dialog)

	dialog.submit()

	if host.handle.closed {
		t.Fatal("failed save closed settings dialog")
	}
	if !strings.Contains(dialog.message.Text(), "disk failed") {
		t.Fatalf("error message = %q", dialog.message.Text())
	}
}

type recordingStore struct {
	loaded  domain.Settings
	saved   domain.Settings
	loadErr error
	saveErr error
}

func (s *recordingStore) Load(context.Context) (domain.Settings, error) {
	return s.loaded, s.loadErr
}

func (s *recordingStore) Save(
	_ context.Context,
	values domain.Settings,
) (domain.Settings, error) {
	s.saved = values
	return values, s.saveErr
}

type testHost struct {
	controls components.Factory
	dialog   modal.Dialog
	handle   *testHandle
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
		handle:   &testHandle{},
	}
}

func (h *testHost) SetFocus(tview.Primitive) {}

func (h *testHost) Refresh() {}

func (h *testHost) Components() components.Factory {
	return h.controls
}

func (h *testHost) OpenModal(dialog modal.Dialog) modal.Handle {
	h.dialog = dialog
	return h.handle
}

type testHandle struct {
	closed bool
}

func (h *testHandle) Close() {
	h.closed = true
}
