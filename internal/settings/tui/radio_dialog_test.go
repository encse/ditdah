package tui

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"ditdah/internal/radio"
	domain "ditdah/internal/settings"
	"ditdah/internal/tui/components"

	"github.com/gdamore/tcell/v2"
)

func TestRadioDialogChecksAndAcceptsConnectionOnOK(t *testing.T) {
	service := &recordingRadioService{
		models: []radio.Model{{
			ID:              3073,
			Manufacturer:    "Icom",
			Name:            "IC-7300",
			DefaultBaudRate: 19200,
		}},
		ports:     []string{"/dev/cu.radio"},
		frequency: 14_074_000,
	}
	host := newTestHost()
	var accepted radio.Config
	var acceptedFrequency uint64
	dialog := newRadioDialog(
		host,
		host.Components(),
		service,
		radio.Config{},
		func(config radio.Config, frequency uint64) {
			accepted = config
			acceptedFrequency = frequency
		},
	)
	handle := &testHandle{}
	dialog.handle = handle
	runRadioDialogUntilLoaded(t, host, dialog)

	if dialog.baudRate.Value() != "19200" {
		t.Fatalf("default baud rate = %q, want 19200", dialog.baudRate.Value())
	}
	dialog.accept()
	if !handle.closed {
		t.Fatal("radio dialog remained open after a successful OK check")
	}
	want := radio.Config{
		ModelID:   3073,
		ModelName: "Icom IC-7300",
		Port:      "/dev/cu.radio",
		BaudRate:  19200,
	}
	if accepted != want || acceptedFrequency != 14_074_000 {
		t.Fatalf("accepted = %#v, %d, want %#v, 14074000", accepted, acceptedFrequency, want)
	}
}

func TestRadioDialogPaintsItsModalBackground(t *testing.T) {
	modalBackground := tcell.NewRGBColor(190, 190, 190)
	host := newTestHost()
	host.controls = components.New(components.Dependencies{
		Theme: components.Theme{Background: tcell.ColorBlack},
		ModalTheme: components.Theme{
			Background:             modalBackground,
			PrimaryText:            tcell.ColorWhite,
			MutedText:              tcell.ColorGray,
			DangerText:             tcell.ColorRed,
			Accent:                 tcell.ColorWhite,
			Border:                 tcell.ColorWhite,
			LabelColor:             tcell.ColorWhite,
			FieldTextColor:         tcell.ColorBlack,
			FieldBackground:        tcell.ColorGray,
			ActiveFieldBackground:  tcell.ColorWhite,
			SelectionText:          tcell.ColorWhite,
			SelectionBackground:    tcell.ColorBlue,
			PopupBorder:            tcell.ColorWhite,
			ButtonText:             tcell.ColorWhite,
			ButtonBackground:       tcell.ColorBlue,
			ActiveButtonText:       tcell.ColorBlack,
			ActiveButtonBackground: tcell.ColorAqua,
		},
	})
	dialog := newRadioDialog(
		host,
		host.Components(),
		&recordingRadioService{},
		radio.Config{},
		nil,
	)
	size := dialog.Size()
	screen := tcell.NewSimulationScreen("UTF-8")
	if err := screen.Init(); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(screen.Fini)
	screen.SetSize(size.Width, size.Height)
	dialog.Content().SetRect(0, 0, size.Width, size.Height)
	dialog.Content().Draw(screen)

	for y := 1; y < size.Height-1; y++ {
		_, _, style, _ := screen.GetContent(2, y)
		_, background, _ := style.Decompose()
		if background != modalBackground {
			t.Fatalf("modal background at row %d = %v, want %v", y, background, modalBackground)
		}
	}
	for _, cell := range []struct{ x, y int }{
		{5, 2}, // radio label area
		{5, 3}, // serial-port label area
		{5, 4}, // baud-rate label area
		{5, 5}, // spacer row
		{5, 6}, // status row
	} {
		_, _, style, _ := screen.GetContent(cell.x, cell.y)
		_, background, _ := style.Decompose()
		if background != modalBackground {
			t.Fatalf(
				"radio content background at (%d, %d) = %v, want %v",
				cell.x,
				cell.y,
				background,
				modalBackground,
			)
		}
	}
}

func TestRadioDialogValidatesFieldsBeforeCheckingOnOK(t *testing.T) {
	service := &recordingRadioService{
		models: []radio.Model{{
			ID: 1, Manufacturer: "Test", Name: "Rig", DefaultBaudRate: 9600,
		}},
		ports: []string{"COM3"},
	}
	host := newTestHost()
	dialog := newRadioDialog(
		host, host.Components(), service, radio.Config{}, nil,
	)
	handle := &testHandle{}
	dialog.handle = handle
	runRadioDialogUntilLoaded(t, host, dialog)

	dialog.baudRate.SetValue("invalid")
	dialog.accept()
	if handle.closed {
		t.Fatal("radio dialog accepted an invalid baud rate")
	}
	if !strings.Contains(dialog.status.Text(), "baud rate") {
		t.Fatalf("validation status = %q", dialog.status.Text())
	}
}

func TestRadioDialogKeepsFailedConnectionOpen(t *testing.T) {
	service := &recordingRadioService{
		models: []radio.Model{{
			ID: 1, Manufacturer: "Test", Name: "Rig", DefaultBaudRate: 9600,
		}},
		ports:    []string{"COM3"},
		checkErr: errors.New("radio did not answer"),
	}
	host := newTestHost()
	dialog := newRadioDialog(
		host, host.Components(), service, radio.Config{}, nil,
	)
	handle := &testHandle{}
	dialog.handle = handle
	runRadioDialogUntilLoaded(t, host, dialog)

	dialog.accept()
	if handle.closed {
		t.Fatal("failed radio connection closed the dialog")
	}
	if !strings.Contains(dialog.status.Text(), "radio did not answer") {
		t.Fatalf("failure status = %q", dialog.status.Text())
	}
}

func TestSettingsChecksStoredRadioWhenOpened(t *testing.T) {
	want := radio.Config{
		ModelID:   3073,
		ModelName: "Icom IC-7300",
		Port:      "/dev/cu.radio",
		BaudRate:  19200,
	}
	service := &recordingRadioService{
		frequency:     7_030_000,
		checkRequests: make(chan radio.Config, 1),
	}
	host := newTestHost()
	settings := newDialog(
		host,
		&recordingStore{loaded: domain.Settings{
			RadioModelID:    want.ModelID,
			RadioModelName:  want.ModelName,
			RadioSerialPort: want.Port,
			RadioBaudRate:   want.BaudRate,
		}},
		&recordingQRZService{},
		nil,
		service,
	)
	if _, loaded := settings.load(t.Context()); !loaded {
		t.Fatal("settings did not load")
	}
	settings.validateStoredRadio(t.Context())
	var checked radio.Config
	select {
	case checked = <-service.checkRequests:
	case <-time.After(time.Second):
		t.Fatal("stored radio was not checked")
	}
	if checked != want {
		t.Fatalf("checked config = %#v, want %#v", checked, want)
	}
	if !strings.Contains(settings.radioStatus.Text(), "7.030 MHz") {
		t.Fatalf("radio status = %q", settings.radioStatus.Text())
	}
}

func TestSettingsStagesTestedRadioUntilMainSettingsAreSaved(t *testing.T) {
	service := &recordingRadioService{
		models: []radio.Model{{
			ID: 3073, Manufacturer: "Icom", Name: "IC-7300", DefaultBaudRate: 19200,
		}},
		ports:     []string{"/dev/cu.radio"},
		frequency: 14_074_000,
	}
	store := &recordingStore{}
	host := newTestHost()
	settings := newDialog(
		host, store, &recordingQRZService{}, nil, service,
	)
	settings.handle = host.OpenModalForCurrentLayer(settings)
	loadSettingsDialog(t, host, settings)

	settings.openRadio()
	editor := host.lastDialog().(*radioDialog)
	runRadioDialogUntilLoaded(t, host, editor)
	editor.accept()

	if store.saveCalls != 0 {
		t.Fatalf("save calls before main OK = %d, want 0", store.saveCalls)
	}
	if !strings.Contains(settings.radioInfo.Text(), "Icom IC-7300") {
		t.Fatalf("radio info = %q", settings.radioInfo.Text())
	}
	settings.submit()
	if store.saved.RadioModelID != 3073 ||
		store.saved.RadioSerialPort != "/dev/cu.radio" ||
		store.saved.RadioBaudRate != 19200 {
		t.Fatalf("saved radio settings = %#v", store.saved)
	}
}

func runRadioDialogUntilLoaded(
	t *testing.T,
	host *testHost,
	dialog *radioDialog,
) {
	t.Helper()
	host.updated = make(chan struct{}, 4)
	ctx, cancel := context.WithCancel(t.Context())
	done := make(chan struct{})
	go func() {
		dialog.Run(ctx)
		close(done)
	}()
	select {
	case <-host.updated:
	case <-time.After(time.Second):
		cancel()
		t.Fatal("radio dialog did not load")
	}
	cancel()
	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("radio dialog Run did not stop")
	}
}

type recordingRadioService struct {
	models        []radio.Model
	ports         []string
	frequency     uint64
	modelsErr     error
	portsErr      error
	checkErr      error
	checkRequests chan radio.Config
}

func (s *recordingRadioService) Models() ([]radio.Model, error) {
	return s.models, s.modelsErr
}

func (s *recordingRadioService) Ports() ([]string, error) {
	return s.ports, s.portsErr
}

func (s *recordingRadioService) Check(
	_ context.Context,
	config radio.Config,
) (uint64, error) {
	if s.checkRequests != nil {
		s.checkRequests <- config
	}
	return s.frequency, s.checkErr
}
