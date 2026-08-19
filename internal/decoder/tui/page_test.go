package tui

import (
	"context"
	"sync"
	"testing"
	"time"

	"morsemanual/internal/audio"
	domain "morsemanual/internal/decoder"
	"morsemanual/internal/settings"
	"morsemanual/internal/tui/components"
	"morsemanual/internal/tui/modal"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

func TestPageMetadata(t *testing.T) {
	page := New(newTestHost(), nil, nil)

	if page.ID() != "morse-decoder" {
		t.Fatalf("ID() = %q, want morse-decoder", page.ID())
	}
	if page.Title() != "Morse decoder" {
		t.Fatalf("Title() = %q, want Morse decoder", page.Title())
	}
	if page.Content() == nil {
		t.Fatal("Content() is nil")
	}
	if len(page.Focusables()) != 1 {
		t.Fatalf("Focusables() = %d items, want decoder output", len(page.Focusables()))
	}
}

func TestPageHasEmptyDecoderOutputAndRightPanel(t *testing.T) {
	page := New(newTestHost(), nil, nil).(*page)
	if page.output.Text() != "" {
		t.Fatalf("output text = %q, want empty", page.output.Text())
	}
	if page.right.Text() != "" {
		t.Fatalf("right panel text = %q, want empty", page.right.Text())
	}
	flex := page.content.(*tview.Flex)
	if flex.GetItemCount() != 2 {
		t.Fatalf("split item count = %d, want 2", flex.GetItemCount())
	}
	if flex.GetItem(0) != page.output || flex.GetItem(1) != page.right {
		t.Fatal("split panels are in the wrong order")
	}
}

func TestPageWritesStreamingDecoderOutput(t *testing.T) {
	device := audio.Device{ID: "radio", Name: "USB radio"}
	source := newRecordingAudioSource(device)
	store := &decoderSettingsStore{values: settings.Settings{
		MorseInputDeviceID: device.ID,
	}}
	page := newPage(
		newTestHost(),
		source,
		store,
		func() (domain.Streaming, error) {
			return emittingStream{text: "CQ "}, nil
		},
	)
	cancel, done := runDecoderPage(t, page)

	waitForAudioStart(t, source.starts, "radio")
	waitForSignal(t, source.consumed, "decoded audio chunk")
	if page.output.Text() != "CQ " {
		t.Fatalf("decoder output = %q, want CQ", page.output.Text())
	}
	if page.Status() != "Listening: USB radio" {
		t.Fatalf("Status() = %q", page.Status())
	}

	cancel()
	waitForRun(t, done)
}

func TestPageRestartsCaptureWhenInputChanges(t *testing.T) {
	first := audio.Device{ID: "first", Name: "First input"}
	second := audio.Device{ID: "second", Name: "Second input"}
	source := newRecordingAudioSource(first, second)
	store := &decoderSettingsStore{values: settings.Settings{
		MorseInputDeviceID: first.ID,
	}}
	page := newPage(
		newTestHost(),
		source,
		store,
		func() (domain.Streaming, error) { return emittingStream{}, nil },
	)
	cancel, done := runDecoderPage(t, page)

	waitForAudioStart(t, source.starts, first.ID)
	store.setInput(second.ID)
	page.inputChanged.Activate()
	waitForAudioStart(t, source.starts, second.ID)
	if page.Status() != "Listening: Second input" {
		t.Fatalf("Status() = %q", page.Status())
	}

	cancel()
	waitForRun(t, done)
}

func TestHiddenPageDefersChangedInputUntilNextActivation(t *testing.T) {
	device := audio.Device{ID: "second", Name: "Second input"}
	source := newRecordingAudioSource(device)
	store := &decoderSettingsStore{values: settings.Settings{
		MorseInputDeviceID: device.ID,
	}}
	page := newPage(
		newTestHost(),
		source,
		store,
		func() (domain.Streaming, error) { return emittingStream{}, nil },
	)

	if len(source.starts) != 0 {
		t.Fatal("hidden decoder started after input change")
	}

	cancel, done := runDecoderPage(t, page)
	waitForAudioStart(t, source.starts, device.ID)
	cancel()
	waitForRun(t, done)
	waitForSignal(t, source.stops, "hidden decoder stop")
}

func TestPageWaitsForInputChangeAfterMissingSelection(t *testing.T) {
	device := audio.Device{ID: "radio", Name: "USB radio"}
	source := newRecordingAudioSource(device)
	store := &decoderSettingsStore{
		loads: make(chan struct{}, 4),
	}
	page := newPage(
		newTestHost(),
		source,
		store,
		func() (domain.Streaming, error) { return emittingStream{}, nil },
	)
	cancel, done := runDecoderPage(t, page)

	waitForSignal(t, store.loads, "missing input settings load")
	store.setInput(device.ID)
	page.inputChanged.Activate()
	waitForAudioStart(t, source.starts, device.ID)

	cancel()
	waitForRun(t, done)
}

func TestPageOwnsMorseInputMenuItem(t *testing.T) {
	page := New(newTestHost(), nil, nil)
	items := page.MenuItems(t.Context())
	if len(items) != 1 {
		t.Fatalf("MenuItems() = %d items, want 1", len(items))
	}
	if items[0].Label != "Morse input" {
		t.Fatalf("menu label = %q, want Morse input", items[0].Label)
	}
}

type emittingStream struct {
	text string
}

func (s emittingStream) Process(
	ctx context.Context,
	_ audio.Chunk,
	consume func(context.Context, string) error,
) error {
	if s.text == "" {
		return nil
	}
	return consume(ctx, s.text)
}

type recordingAudioSource struct {
	devices  []audio.Device
	starts   chan string
	consumed chan struct{}
	stops    chan struct{}
}

func newRecordingAudioSource(devices ...audio.Device) *recordingAudioSource {
	return &recordingAudioSource{
		devices:  devices,
		starts:   make(chan string, 8),
		consumed: make(chan struct{}, 8),
		stops:    make(chan struct{}, 8),
	}
}

func (s *recordingAudioSource) Devices() ([]audio.Device, error) {
	return append([]audio.Device(nil), s.devices...), nil
}

func (s *recordingAudioSource) Run(
	ctx context.Context,
	device audio.Device,
	consume func(context.Context, audio.Chunk) error,
) error {
	s.starts <- device.ID
	if err := consume(ctx, audio.Chunk{}); err != nil {
		return err
	}
	s.consumed <- struct{}{}
	<-ctx.Done()
	s.stops <- struct{}{}
	return ctx.Err()
}

func (s *recordingAudioSource) Close() error { return nil }

type decoderSettingsStore struct {
	mu     sync.Mutex
	values settings.Settings
	loads  chan struct{}
}

func (s *decoderSettingsStore) Load(context.Context) (settings.Settings, error) {
	s.mu.Lock()
	values := s.values
	s.mu.Unlock()
	if s.loads != nil {
		s.loads <- struct{}{}
	}
	return values, nil
}

func (s *decoderSettingsStore) Save(
	_ context.Context,
	values settings.Settings,
) (settings.Settings, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.values = values
	return values, nil
}

func (s *decoderSettingsStore) setInput(id string) {
	s.mu.Lock()
	s.values.MorseInputDeviceID = id
	s.mu.Unlock()
}

func waitForAudioStart(t *testing.T, starts <-chan string, want string) {
	t.Helper()
	select {
	case got := <-starts:
		if got != want {
			t.Fatalf("audio started on %q, want %q", got, want)
		}
	case <-time.After(time.Second):
		t.Fatalf("audio did not start on %q", want)
	}
}

func waitForSignal(t *testing.T, signal <-chan struct{}, description string) {
	t.Helper()
	select {
	case <-signal:
	case <-time.After(time.Second):
		t.Fatalf("timed out waiting for %s", description)
	}
}

func runDecoderPage(
	t *testing.T,
	page *page,
) (context.CancelFunc, <-chan struct{}) {
	t.Helper()
	ctx, cancel := context.WithCancel(t.Context())
	done := make(chan struct{})
	go func() {
		defer close(done)
		page.Run(ctx)
	}()
	return cancel, done
}

func waitForRun(t *testing.T, done <-chan struct{}) {
	t.Helper()
	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("decoder page Run() did not return")
	}
}

type testHost struct {
	controls components.Factory
}

func newTestHost() testHost {
	theme := components.Theme{
		Background:     tcell.ColorBlack,
		PrimaryText:    tcell.ColorWhite,
		MutedText:      tcell.ColorGray,
		FieldTextColor: tcell.ColorWhite,
	}
	return testHost{controls: components.New(components.Dependencies{Theme: theme})}
}

func (h testHost) SetFocus(tview.Primitive) {}

func (h testHost) Refresh() {}

func (h testHost) Update(update func()) {
	if update != nil {
		update()
	}
}

func (h testHost) Components() components.Factory { return h.controls }

func (h testHost) OpenModal(modal.Dialog) modal.Handle { return nil }
