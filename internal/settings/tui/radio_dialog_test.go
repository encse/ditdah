package tui

import (
	"context"
	"errors"
	"strings"
	"sync"
	"testing"
	"time"

	"ditdah/internal/radio"
	"ditdah/internal/syncutil"
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

func TestSettingsDisplaysRadioUpdatesAndUnsubscribesOnStop(t *testing.T) {
	service := &recordingRadioService{
		status:             radio.Status{Error: "radio did not answer"},
		subscriptionClosed: make(chan struct{}),
	}
	host := newTestHost()
	settings := newDialog(
		host,
		&recordingStore{},
		&recordingQRZService{},
		nil,
		service,
	)
	host.updated = make(chan struct{}, 2)
	ctx, cancel := context.WithCancel(t.Context())
	done := make(chan struct{})
	go func() {
		settings.runRadioStatus(ctx)
		close(done)
	}()
	select {
	case <-host.updated:
	case <-time.After(time.Second):
		t.Fatal("settings did not display the subscribed radio status")
	}
	if !strings.Contains(settings.radioInfo.Text(), "radio did not answer") {
		t.Fatalf("radio error = %q", settings.radioInfo.Text())
	}
	service.setStatus(radio.Status{FrequencyHz: 7_030_000})
	select {
	case <-host.updated:
	case <-time.After(time.Second):
		t.Fatal("settings did not display the changed radio status")
	}
	if settings.radioInfo.Text() != "7.030 MHz" {
		t.Fatalf("radio info = %q, want 7.030 MHz", settings.radioInfo.Text())
	}
	cancel()
	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("settings Run did not stop")
	}
	select {
	case <-service.subscriptionClosed:
	default:
		t.Fatal("settings did not close its radio subscription")
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
	if settings.radioInfo.Text() != "14.074 MHz" {
		t.Fatalf("radio info = %q, want 14.074 MHz", settings.radioInfo.Text())
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
	models             []radio.Model
	ports              []string
	frequency          uint64
	modelsErr          error
	portsErr           error
	checkErr           error
	checkRequests      chan radio.Config
	status             radio.Status
	changes            syncutil.Broadcaster
	mu                 sync.Mutex
	subscriptionClosed chan struct{}
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

func (s *recordingRadioService) Run(ctx context.Context) { <-ctx.Done() }

func (s *recordingRadioService) Status() radio.Status {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.status == (radio.Status{}) {
		return radio.Status{Error: "Radio is not configured"}
	}
	return s.status
}

func (s *recordingRadioService) Subscribe() syncutil.Subscription {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.changes == nil {
		s.changes = syncutil.NewBroadcaster()
	}
	subscription := s.changes.Subscribe()
	if s.subscriptionClosed == nil {
		return subscription
	}
	return &closingSubscription{
		Subscription: subscription,
		closed:       s.subscriptionClosed,
	}
}

func (s *recordingRadioService) setStatus(status radio.Status) {
	s.mu.Lock()
	s.status = status
	changes := s.changes
	s.mu.Unlock()
	if changes != nil {
		changes.Activate()
	}
}

type closingSubscription struct {
	syncutil.Subscription
	closed chan struct{}
	once   sync.Once
}

func (s *closingSubscription) Close() {
	s.Subscription.Close()
	s.once.Do(func() { close(s.closed) })
}
