// Package tui implements the Morse decoder terminal user interface.
package tui

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"ditdah/internal/audio"
	"ditdah/internal/callsign"
	domain "ditdah/internal/decoder"
	"ditdah/internal/optional"
	"ditdah/internal/radio"
	"ditdah/internal/settings"
	"ditdah/internal/syncutil"
	ui "ditdah/internal/tui"
	"ditdah/internal/tui/components"
	"ditdah/internal/tui/keybinding"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
	"golang.org/x/sync/errgroup"
)

var errSettingsChanged = errors.New("settings changed")
var errAudioStopped = errors.New("audio input stopped")

type streamFactory func() (domain.Streaming, error)

type lookupRequest struct {
	generation uint64
	callsign   string
}

type decoderState struct {
	callsigns        []string
	selectedCallsign string
	decodedText      strings.Builder
}

type page struct {
	*decoderState
	host             ui.PageHost
	source           audio.Source
	settings         settings.Store
	radio            radio.StatusSource
	lookup           callsign.Service
	newStream        streamFactory
	content          tview.Primitive
	output           components.TextView
	right            tview.Primitive
	callsignList     components.Table
	details          components.TextView
	audioStatus      string
	radioStatus      string
	lookups          syncutil.Mailbox[lookupRequest]
	showNewQSOEditor func(ui.Owner, string)

	lookupGeneration uint64
}

// New creates the page for live Morse decoder output.
func New(
	host ui.PageHost,
	source audio.Source,
	settingsStore settings.Store,
	lookup callsign.Service,
	showNewQSOEditor func(ui.Owner, string),
	radioSource radio.StatusSource,
) ui.Page {
	return newPage(
		host,
		source,
		settingsStore,
		lookup,
		showNewQSOEditor,
		domain.NewStreaming,
		radioSource,
	)
}

// NewFactory creates fresh decoder pages which restore the current decoder
// session state whenever the user navigates back to them.
func NewFactory(
	host ui.PageHost,
	source audio.Source,
	settingsStore settings.Store,
	lookup callsign.Service,
	showNewQSOEditor func(ui.Owner, string),
	radioSource radio.StatusSource,
) func() ui.Page {
	state := &decoderState{}
	return func() ui.Page {
		return newPageWithState(
			host,
			source,
			settingsStore,
			lookup,
			showNewQSOEditor,
			state,
			domain.NewStreaming,
			radioSource,
		)
	}
}

func newPage(
	host ui.PageHost,
	source audio.Source,
	settingsStore settings.Store,
	lookup callsign.Service,
	showNewQSOEditor func(ui.Owner, string),
	newStream streamFactory,
	radioSource radio.StatusSource,
) *page {
	return newPageWithState(
		host,
		source,
		settingsStore,
		lookup,
		showNewQSOEditor,
		&decoderState{},
		newStream,
		radioSource,
	)
}

func newPageWithState(
	host ui.PageHost,
	source audio.Source,
	settingsStore settings.Store,
	lookup callsign.Service,
	showNewQSOEditor func(ui.Owner, string),
	state *decoderState,
	newStream streamFactory,
	radioSource radio.StatusSource,
) *page {
	controls := host.Components()
	page := &page{
		decoderState:     state,
		host:             host,
		source:           source,
		settings:         settingsStore,
		radio:            radioSource,
		lookup:           lookup,
		newStream:        newStream,
		lookups:          syncutil.NewMailbox(lookupRequest{}),
		showNewQSOEditor: showNewQSOEditor,
		audioStatus:      "Paused",
	}
	output := controls.TextView()
	output.SetStyle(components.TextViewPrimary)
	output.SetBorder(" Decoded text ")
	output.SetScrollable(true)
	output.SetWrap(true)
	output.SetWordWrap(true)
	output.SetRegions(true)
	output.Highlight(decodedCallsignRegion)
	output.ScrollToEnd()

	page.output = output
	page.callsignList = page.newCallsignList(controls)
	page.details = controls.TextView()
	page.details.SetStyle(components.TextViewPrimary)
	page.details.SetBorder(" QRZ.com ")
	page.details.SetScrollable(true)
	page.details.SetWrap(true)
	page.details.SetWordWrap(true)
	page.renderCallsigns()
	page.renderDecodedText()

	right := controls.Flex(tview.FlexRow).
		AddItem(page.callsignList, 0, 1, true).
		AddItem(page.details, 0, 1, false)
	page.right = right

	content := controls.Flex(tview.FlexColumn).
		AddItem(output, 0, 2, true).
		AddItem(right, 0, 1, false)
	page.content = content
	return page
}

func (p *page) ID() string { return "morse-decoder" }

func (p *page) Title() string { return "Morse decoder" }

func (p *page) Content() tview.Primitive { return p.content }

func (p *page) Focusables() []tview.Primitive {
	return []tview.Primitive{p.output, p.callsignList, p.details}
}

func (p *page) KeyBindings() []keybinding.Binding {
	return []keybinding.Binding{
		keybinding.OnRune('a', "add callsign", p.openAddCallsign),
		keybinding.OnKey(tcell.KeyEnter, "new QSO", p.openCreateQSO),
		keybinding.OnRune('d', "delete callsign", p.confirmDeleteSelectedCallsign),
		keybinding.OnRune('c', "clear log", p.confirmClearLog),
	}
}

func (p *page) MenuItems() []components.MenuItem { return nil }

func (p *page) Status() string {
	if p.radioStatus == "" {
		return p.audioStatus
	}
	return p.radioStatus + "  " + p.audioStatus
}

// Run decodes audio while the page is visible. Input changes end only the
// current audio session; the page run continues with the saved input.
func (p *page) Run(ctx context.Context) {
	// Re-issue the current selection on every activation. If a lookup was
	// cancelled when the page was hidden, the next Run retries it without
	// retaining cross-run goroutines or reading UI state off the event loop.
	p.host.Update(p, p.requestSelectedCallsign)
	var group errgroup.Group
	group.Go(func() error {
		p.runDecoder(ctx)
		return nil
	})
	group.Go(func() error {
		p.runLookups(ctx)
		return nil
	})
	group.Go(func() error {
		p.runRadioStatus(ctx)
		return nil
	})
	_ = group.Wait()
}

func (p *page) runRadioStatus(ctx context.Context) {
	if p.radio == nil {
		return
	}
	changes := p.radio.Subscribe()
	defer changes.Close()
	for ctx.Err() == nil {
		status := p.radio.Status()
		p.host.Update(p, func() {
			p.radioStatus = ""
			if status.Error == "" && status.FrequencyHz > 0 {
				p.radioStatus = ui.FormatFrequencyMHz(status.FrequencyHz)
			}
		})
		if err := changes.Wait(ctx); err != nil {
			return
		}
	}
}

func (p *page) runDecoder(ctx context.Context) {
	if p.settings == nil {
		p.setStatus("Error: settings are unavailable")
		return
	}
	settingsChanges := p.settings.Subscribe()
	defer settingsChanges.Close()

	for ctx.Err() == nil {
		p.setStatus("Starting decoder...")
		err := p.runSession(ctx, settingsChanges)
		if ctx.Err() != nil {
			return
		}
		if errors.Is(err, errSettingsChanged) {
			continue
		}
		if err == nil {
			err = errAudioStopped
		}
		p.setStatus("Error: " + err.Error())
		if err := settingsChanges.Wait(ctx); err != nil {
			return
		}
	}
}

func (p *page) runLookups(ctx context.Context) {
	for {
		request, err := p.lookups.Receive(ctx)
		if err != nil {
			return
		}
		if request.callsign == "" {
			continue
		}
		if p.lookup == nil {
			p.showLookupResult(request, callsign.Entry{}, errors.New(
				"callsign lookup is unavailable",
			))
			continue
		}
		entry, lookupErr := p.lookup.Lookup(ctx, request.callsign)
		if ctx.Err() != nil {
			return
		}
		p.showLookupResult(request, entry, lookupErr)
	}
}

func (p *page) showLookupResult(
	request lookupRequest,
	entry callsign.Entry,
	err error,
) {
	p.host.Update(p, func() {
		if request.generation != p.lookupGeneration ||
			request.callsign != p.selectedCallsign {
			return
		}
		if err != nil {
			p.details.SetStyle(components.TextViewDanger)
			p.details.SetText("Error: " + err.Error())
			return
		}
		p.details.SetStyle(components.TextViewPrimary)
		p.details.SetText(formatCallsignDetails(entry))
	})
}

func (p *page) openCreateQSO() {
	if p.selectedCallsign == "" || p.showNewQSOEditor == nil {
		return
	}
	p.showNewQSOEditor(p, p.selectedCallsign)
}

func formatCallsignDetails(entry callsign.Entry) string {
	if entry.Status == callsign.StatusError {
		return strings.TrimSpace(entry.Error)
	}
	record, present := entry.Record.Get()
	if !present {
		return "No QRZ.com details available."
	}
	lines := []string{"Callsign: " + record.Callsign}
	appendOptionalDetail := func(label string, value optional.Value[string]) {
		if text, ok := value.Get(); ok && strings.TrimSpace(text) != "" {
			lines = append(lines, label+": "+text)
		}
	}
	appendOptionalDetail("Name", record.Name)
	appendOptionalDetail("Nickname", record.Nickname)
	appendOptionalDetail("QTH", record.QTH)
	appendOptionalDetail("State", record.State)
	appendOptionalDetail("Country", record.Country)
	appendOptionalDetail("Grid", record.Grid)
	appendOptionalDetail("CQ zone", record.CQZone)
	appendOptionalDetail("ITU zone", record.ITUZone)
	appendOptionalDetail("QRZ", record.QRZURL)
	return strings.Join(lines, "\n")
}

func (p *page) runSession(
	ctx context.Context,
	settingsChanges syncutil.Subscription,
) error {
	if p.source == nil {
		return errors.New("audio input is unavailable")
	}
	if p.settings == nil {
		return errors.New("settings are unavailable")
	}

	configured, err := p.settings.Load(ctx)
	if err != nil {
		return fmt.Errorf("load Morse input setting: %w", err)
	}
	if configured.MorseInputDeviceID == "" {
		return errors.New("select a Morse input in Settings")
	}

	devices, err := p.source.Devices()
	if err != nil {
		return err
	}
	device, found := findDevice(devices, configured.MorseInputDeviceID)
	if !found {
		return fmt.Errorf(
			"%w: %s",
			audio.ErrDeviceNotFound,
			configured.MorseInputDeviceID,
		)
	}

	stream, err := p.newStream()
	if err != nil {
		return fmt.Errorf("load decoder model: %w", err)
	}
	p.setStatus("Listening: " + device.Name)

	group, sessionCtx := errgroup.WithContext(ctx)
	group.Go(func() error {
		err := p.source.Run(sessionCtx, device, func(
			ctx context.Context,
			chunk audio.Chunk,
		) error {
			return stream.Process(ctx, chunk, p.appendDecoded)
		})
		if err == nil {
			return errAudioStopped
		}
		return err
	})
	group.Go(func() error {
		if err := settingsChanges.Wait(sessionCtx); err != nil {
			return err
		}
		return errSettingsChanged
	})
	return group.Wait()
}

func (p *page) appendDecoded(ctx context.Context, text string) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	p.host.Update(p, func() {
		wasAtEnd := p.output.AtEnd()
		previousLength := p.decodedText.Len()
		_, _ = p.decodedText.WriteString(text)
		if p.newDecodedMatch(previousLength) {
			p.renderDecodedText()
		} else {
			_, _ = p.output.Write([]byte(tview.Escape(text)))
		}
		if wasAtEnd {
			p.output.ScrollToEnd()
		}
	})
	return nil
}

func (p *page) setStatus(status string) {
	p.host.Update(p, func() {
		p.audioStatus = status
	})
}

func findDevice(devices []audio.Device, id string) (audio.Device, bool) {
	for _, device := range devices {
		if device.ID == id {
			return device, true
		}
	}
	return audio.Device{}, false
}
