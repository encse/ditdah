package tui

import (
	"errors"
	"strings"
	"testing"

	"morsemanual/internal/audio"
	domain "morsemanual/internal/settings"
)

func TestOpenMorseInputLoadsDevicesAndSavedSelection(t *testing.T) {
	store := &recordingStore{loaded: domain.Settings{
		MorseInputDeviceID: "usb",
	}}
	host := newTestHost()
	inputs := recordingDeviceLister{devices: []audio.Device{
		{ID: "built-in", Name: "Built-in input", IsDefault: true},
		{ID: "usb", Name: "USB radio"},
	}}

	OpenMorseInput(t.Context(), host, inputs, store, nil)
	dialog := host.lastDialog().(*morseInputDialog)

	index, label := dialog.input.CurrentOption()
	if index != 1 || label != "USB radio" {
		t.Fatalf("selected input = %d, %q, want saved USB device", index, label)
	}
	if dialog.handle == nil {
		t.Fatal("input dialog has no modal handle")
	}
}

func TestMorseInputDefaultsToSystemDefaultDevice(t *testing.T) {
	dialog := newMorseInputDialog(
		t.Context(),
		newTestHost(),
		&recordingStore{},
		domain.Settings{},
		[]audio.Device{
			{ID: "first", Name: "Line input"},
			{ID: "default", Name: "USB radio", IsDefault: true},
		},
		nil,
	)

	index, label := dialog.input.CurrentOption()
	if index != 1 || label != "USB radio (default)" {
		t.Fatalf("selected input = %d, %q, want default device", index, label)
	}
}

func TestMorseInputKeepsUnavailableSelectionUnselected(t *testing.T) {
	dialog := newMorseInputDialog(
		t.Context(),
		newTestHost(),
		&recordingStore{},
		domain.Settings{MorseInputDeviceID: "disconnected"},
		[]audio.Device{{ID: "available", Name: "Built-in input", IsDefault: true}},
		nil,
	)
	dialog.showInitialError(nil)

	index, _ := dialog.input.CurrentOption()
	if index != -1 {
		t.Fatalf("selected input = %d, want no silent replacement", index)
	}
	if !strings.Contains(dialog.message.Text(), "unavailable") {
		t.Fatalf("message = %q, want unavailable input error", dialog.message.Text())
	}
}

func TestMorseInputSubmitPersistsDeviceIDAndPreservesSettings(t *testing.T) {
	values := domain.Settings{
		StationCallsign: "HA7NCS",
		QRZAPIKey:       "qrz-key",
	}
	store := &recordingStore{loaded: values}
	host := newTestHost()
	reloads := 0
	dialog := newMorseInputDialog(
		t.Context(),
		host,
		store,
		values,
		[]audio.Device{{ID: "capture-id", Name: "USB radio"}},
		func() { reloads++ },
	)
	dialog.handle = host.OpenModal(dialog)

	dialog.submit()

	want := values
	want.MorseInputDeviceID = "capture-id"
	if store.saved != want {
		t.Fatalf("saved settings = %#v, want %#v", store.saved, want)
	}
	if !host.lastHandle().closed {
		t.Fatal("successful input selection did not close dialog")
	}
	if reloads != 1 {
		t.Fatalf("decoder reloads = %d, want 1", reloads)
	}
}

func TestMorseInputSubmitRequiresAvailableSelection(t *testing.T) {
	store := &recordingStore{}
	dialog := newMorseInputDialog(
		t.Context(),
		newTestHost(),
		store,
		domain.Settings{},
		nil,
		nil,
	)

	dialog.submit()

	if store.saveCalls != 0 {
		t.Fatalf("save calls = %d, want 0", store.saveCalls)
	}
	if !strings.Contains(dialog.message.Text(), "select an available") {
		t.Fatalf("message = %q, want selection error", dialog.message.Text())
	}
}

type recordingDeviceLister struct {
	devices []audio.Device
	err     error
}

func (l recordingDeviceLister) Devices() ([]audio.Device, error) {
	return l.devices, l.err
}

func TestOpenMorseInputShowsDeviceEnumerationError(t *testing.T) {
	host := newTestHost()
	OpenMorseInput(
		t.Context(),
		host,
		recordingDeviceLister{err: errors.New("capture backend failed")},
		&recordingStore{},
		nil,
	)
	dialog := host.lastDialog().(*morseInputDialog)
	if !strings.Contains(dialog.message.Text(), "capture backend failed") {
		t.Fatalf("message = %q, want device enumeration error", dialog.message.Text())
	}
}
