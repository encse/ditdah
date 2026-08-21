package tui

import (
	"context"
	"strings"
	"sync"
	"testing"
	"time"

	"morsemanual/internal/audio"
	"morsemanual/internal/callsign"
	domain "morsemanual/internal/decoder"
	logbookdomain "morsemanual/internal/logbook"
	"morsemanual/internal/optional"
	"morsemanual/internal/settings"
	ui "morsemanual/internal/tui"
	"morsemanual/internal/tui/components"
	"morsemanual/internal/tui/modal"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

func TestPageMetadata(t *testing.T) {
	page := New(newTestHost(), nil, nil, nil, nil)

	if page.ID() != "morse-decoder" {
		t.Fatalf("ID() = %q, want morse-decoder", page.ID())
	}
	if page.Title() != "Morse decoder" {
		t.Fatalf("Title() = %q, want Morse decoder", page.Title())
	}
	if page.Content() == nil {
		t.Fatal("Content() is nil")
	}
	if len(page.Focusables()) != 3 {
		t.Fatalf("Focusables() = %d items, want output, callsigns, details", len(page.Focusables()))
	}
}

func TestPageHasEmptyDecoderOutputAndRightPanel(t *testing.T) {
	page := New(newTestHost(), nil, nil, nil, nil).(*page)
	if page.output.Text() != "" {
		t.Fatalf("output text = %q, want empty", page.output.Text())
	}
	if page.details.Text() != "" {
		t.Fatalf("details text = %q, want empty", page.details.Text())
	}
	flex := page.content.(*tview.Flex)
	if flex.GetItemCount() != 2 {
		t.Fatalf("split item count = %d, want 2", flex.GetItemCount())
	}
	if flex.GetItem(0) != page.output || flex.GetItem(1) != page.right {
		t.Fatal("split panels are in the wrong order")
	}
	right := page.right.(*tview.Flex)
	if right.GetItemCount() != 2 || right.GetItem(0) != page.callsignList ||
		right.GetItem(1) != page.details {
		t.Fatal("right panel must contain callsigns above QRZ details")
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
		nil,
		nil,
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

func TestPageRestartsCaptureWhenSettingsChange(t *testing.T) {
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
		nil,
		nil,
		func() (domain.Streaming, error) { return emittingStream{}, nil },
	)
	cancel, done := runDecoderPage(t, page)

	waitForAudioStart(t, source.starts, first.ID)
	store.setInput(second.ID)
	page.SettingsChanged()
	waitForAudioStart(t, source.starts, second.ID)
	if page.Status() != "Listening: Second input" {
		t.Fatalf("Status() = %q", page.Status())
	}

	cancel()
	waitForRun(t, done)
}

func TestHiddenPageDoesNotReactToSettingsUntilNextActivation(t *testing.T) {
	device := audio.Device{ID: "second", Name: "Second input"}
	source := newRecordingAudioSource(device)
	store := &decoderSettingsStore{values: settings.Settings{
		MorseInputDeviceID: device.ID,
	}}
	page := newPage(
		newTestHost(),
		source,
		store,
		nil,
		nil,
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
		nil,
		nil,
		func() (domain.Streaming, error) { return emittingStream{}, nil },
	)
	cancel, done := runDecoderPage(t, page)

	waitForSignal(t, store.loads, "missing input settings load")
	store.setInput(device.ID)
	page.SettingsChanged()
	waitForAudioStart(t, source.starts, device.ID)

	cancel()
	waitForRun(t, done)
}

func TestPageDoesNotContributeMenuItems(t *testing.T) {
	page := New(newTestHost(), nil, nil, nil, nil)
	items := page.MenuItems()
	if len(items) != 0 {
		t.Fatalf("MenuItems() = %#v, want none", items)
	}
}

func TestPageAddsSelectsAndDeletesCallsigns(t *testing.T) {
	host := newTestHost()
	store := &decoderSettingsStore{values: settings.Settings{
		StationCallsign: "HA7NCS",
	}}
	editors := &recordingQSOEditors{}
	page := New(
		host,
		nil,
		store,
		nil,
		editors.Create,
	).(*page)
	bindings := page.KeyBindings()
	if len(bindings) != 4 || bindings[0].Hint().Keys != "a" ||
		bindings[1].Hint().Keys != "Enter" ||
		bindings[2].Hint().Keys != "d" || bindings[3].Hint().Keys != "c" {
		t.Fatalf("KeyBindings() = %#v, want a/add, Enter/new QSO, d/delete and c/clear", bindings)
	}
	if err := page.addCallsign(" dl1abc "); err != nil {
		t.Fatalf("addCallsign() error = %v", err)
	}
	if err := page.addCallsign("ha7ncs"); err != nil {
		t.Fatalf("addCallsign() error = %v", err)
	}
	if got := strings.Join(page.callsigns, ","); got != "DL1ABC,HA7NCS" {
		t.Fatalf("callsigns = %q", got)
	}
	if page.selectedCallsign != "HA7NCS" {
		t.Fatalf("selected callsign = %q, want HA7NCS", page.selectedCallsign)
	}
	if row, _ := page.callsignList.Selection(); row != 1 {
		t.Fatalf("selected table row = %d, want second data row 1", row)
	}
	if err := page.addCallsign("DL1ABC"); err == nil {
		t.Fatal("duplicate callsign was added")
	}
	if !bindings[1].Handle(tcell.NewEventKey(tcell.KeyEnter, 0, 0)) {
		t.Fatal("Enter new QSO binding was not handled")
	}
	if editors.createdOwner != page.Content() || editors.createdCallsign != "HA7NCS" {
		t.Fatalf("create editor call = %p, %q", editors.createdOwner, editors.createdCallsign)
	}
	if err := page.updateCallsign("HA7NCS", " oe1xyz "); err != nil {
		t.Fatalf("updateCallsign() error = %v", err)
	}
	if page.callsigns[1] != "OE1XYZ" || page.selectedCallsign != "OE1XYZ" {
		t.Fatalf("callsigns = %#v, selected = %q", page.callsigns, page.selectedCallsign)
	}

	page.deleteSelectedCallsign()
	if len(page.callsigns) != 1 || page.callsigns[0] != "DL1ABC" ||
		page.selectedCallsign != "DL1ABC" {
		t.Fatalf(
			"callsigns = %#v, selected = %q",
			page.callsigns,
			page.selectedCallsign,
		)
	}
	if row, _ := page.callsignList.Selection(); row != 0 {
		t.Fatalf("selected table row after delete = %d, want first data row 0", row)
	}
}

func TestDeleteCallsignBindingRequiresConfirmation(t *testing.T) {
	host := newTestHost()
	page := New(host, nil, nil, nil, nil).(*page)
	if err := page.addCallsign("DL1ABC"); err != nil {
		t.Fatal(err)
	}

	if !page.KeyBindings()[2].Handle(
		tcell.NewEventKey(tcell.KeyRune, 'd', 0),
	) {
		t.Fatal("d delete binding was not handled")
	}
	dialog, ok := host.opened.(*confirmDialog)
	if !ok {
		t.Fatalf("d opened %T, want *confirmDialog", host.opened)
	}
	if len(page.callsigns) != 1 {
		t.Fatal("opening delete confirmation changed the callsign list")
	}
	if !strings.Contains(dialog.message.Text(), "DL1ABC") {
		t.Fatalf("confirmation message = %q, want callsign", dialog.message.Text())
	}
	dialog.confirm.InputHandler()(
		tcell.NewEventKey(tcell.KeyEnter, 0, 0),
		nil,
	)
	if len(page.callsigns) != 0 || page.selectedCallsign != "" {
		t.Fatalf("callsigns after delete = %#v, selected = %q", page.callsigns, page.selectedCallsign)
	}
}

func TestPageLooksUpSelectedCallsignDuringRun(t *testing.T) {
	host := newTestHost()
	host.updated = make(chan struct{}, 8)
	lookup := &decoderLookupService{
		requests: make(chan string, 1),
		release:  make(chan struct{}),
		entry: callsign.Entry{
			Callsign: "DL1ABC",
			Status:   callsign.StatusReady,
			Record: optional.Some(callsign.Record{
				Callsign: "DL1ABC",
				Name:     optional.Some("Jane Doe"),
				Country:  optional.Some("Germany"),
				Grid:     optional.Some("JO62"),
			}),
		},
	}
	page := New(host, nil, nil, lookup, nil).(*page)
	ctx, cancel := context.WithCancel(t.Context())
	done := make(chan struct{})
	go func() {
		defer close(done)
		page.runLookups(ctx)
	}()

	if err := page.addCallsign("dl1abc"); err != nil {
		t.Fatal(err)
	}
	select {
	case got := <-lookup.requests:
		if got != "DL1ABC" {
			t.Fatalf("lookup callsign = %q, want DL1ABC", got)
		}
	case <-time.After(time.Second):
		t.Fatal("callsign lookup did not start")
	}
	close(lookup.release)
	waitForSignal(t, host.updated, "callsign details update")
	if details := page.details.Text(); !strings.Contains(details, "Name: Jane Doe") ||
		!strings.Contains(details, "Country: Germany") ||
		!strings.Contains(details, "Grid: JO62") {
		t.Fatalf("details = %q", details)
	}

	cancel()
	waitForSignal(t, done, "callsign lookup worker stop")
}

func TestPageRetriesSelectedCallsignAfterReactivation(t *testing.T) {
	lookup := &cancelingLookupService{requests: make(chan string, 2)}
	page := New(newTestHost(), nil, nil, lookup, nil).(*page)
	if err := page.addCallsign("DL1ABC"); err != nil {
		t.Fatal(err)
	}

	cancel, done := runDecoderPage(t, page)
	waitForLookupRequest(t, lookup.requests, "DL1ABC")
	cancel()
	waitForRun(t, done)

	cancel, done = runDecoderPage(t, page)
	waitForLookupRequest(t, lookup.requests, "DL1ABC")
	cancel()
	waitForRun(t, done)
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
	updated  chan struct{}
	opened   modal.Dialog
}

type recordingQSOEditors struct {
	createdOwner    tview.Primitive
	createdCallsign string
}

func (f *recordingQSOEditors) Create(owner tview.Primitive, callsign string) {
	f.createdOwner = owner
	f.createdCallsign = callsign
}

func (f *recordingQSOEditors) Edit(tview.Primitive, logbookdomain.QSO) {}

func newTestHost() *testHost {
	theme := components.Theme{
		Background:     tcell.ColorBlack,
		PrimaryText:    tcell.ColorWhite,
		MutedText:      tcell.ColorGray,
		FieldTextColor: tcell.ColorWhite,
	}
	modalTheme := theme
	modalTheme.Background = tcell.NewRGBColor(190, 190, 190)
	modalTheme.Border = tcell.ColorWhite
	modalTheme.Accent = tcell.ColorWhite
	return &testHost{controls: components.New(components.Dependencies{
		Theme:      theme,
		ModalTheme: modalTheme,
	})}
}

func (h testHost) SetFocus(tview.Primitive) {}

func (h testHost) Refresh() {}

func (h testHost) Update(update func()) {
	if update != nil {
		update()
	}
	if h.updated != nil {
		h.updated <- struct{}{}
	}
}

func (h testHost) Components() components.Factory { return h.controls }

func (h *testHost) OpenModal(
	_ tview.Primitive,
	dialog modal.Dialog,
) modal.Handle {
	h.opened = dialog
	return nil
}

func (h *testHost) Background(
	_ tview.Primitive,
	_ ui.BackgroundWork,
) bool {
	return false
}

type decoderLookupService struct {
	requests chan string
	release  chan struct{}
	entry    callsign.Entry
}

type cancelingLookupService struct {
	requests chan string
}

func (s *cancelingLookupService) Lookup(
	ctx context.Context,
	value string,
) (callsign.Entry, error) {
	s.requests <- value
	<-ctx.Done()
	return callsign.Entry{}, ctx.Err()
}

func waitForLookupRequest(t *testing.T, requests <-chan string, want string) {
	t.Helper()
	select {
	case got := <-requests:
		if got != want {
			t.Fatalf("lookup callsign = %q, want %q", got, want)
		}
	case <-time.After(time.Second):
		t.Fatalf("callsign lookup did not start for %q", want)
	}
}

func (s *decoderLookupService) Lookup(
	ctx context.Context,
	value string,
) (callsign.Entry, error) {
	s.requests <- value
	select {
	case <-ctx.Done():
		return callsign.Entry{}, ctx.Err()
	case <-s.release:
		return s.entry, nil
	}
}
