// Package tui implements the Morse decoder terminal user interface.
package tui

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"morsemanual/internal/audio"
	"morsemanual/internal/callsign"
	domain "morsemanual/internal/decoder"
	"morsemanual/internal/mailbox"
	"morsemanual/internal/optional"
	"morsemanual/internal/settings"
	"morsemanual/internal/trigger"
	ui "morsemanual/internal/tui"
	"morsemanual/internal/tui/components"
	"morsemanual/internal/tui/keybinding"

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

type page struct {
	host             ui.PageHost
	source           audio.Source
	settings         settings.Store
	lookup           callsign.Service
	newStream        streamFactory
	content          tview.Primitive
	output           components.TextView
	right            tview.Primitive
	callsignList     components.Table
	details          components.TextView
	statusText       string
	settingsChanged  trigger.Trigger
	lookups          mailbox.Mailbox[lookupRequest]
	showNewQSOEditor func(tview.Primitive, string)

	callsigns        []string
	selectedCallsign string
	lookupGeneration uint64
	decodedText      strings.Builder
}

// New creates the page for live Morse decoder output.
func New(
	host ui.PageHost,
	source audio.Source,
	settingsStore settings.Store,
	lookup callsign.Service,
	showNewQSOEditor func(tview.Primitive, string),
) ui.Page {
	return newPage(
		host,
		source,
		settingsStore,
		lookup,
		showNewQSOEditor,
		domain.NewStreaming,
	)
}

func newPage(
	host ui.PageHost,
	source audio.Source,
	settingsStore settings.Store,
	lookup callsign.Service,
	showNewQSOEditor func(tview.Primitive, string),
	newStream streamFactory,
) *page {
	controls := host.Components()
	page := &page{
		host:             host,
		source:           source,
		settings:         settingsStore,
		lookup:           lookup,
		newStream:        newStream,
		settingsChanged:  trigger.New(),
		lookups:          mailbox.New(lookupRequest{}),
		showNewQSOEditor: showNewQSOEditor,
		statusText:       "Paused",
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

func (p *page) SettingsChanged() { p.settingsChanged.Activate() }

func (p *page) Status() string { return p.statusText }

// Run decodes audio while the page is visible. Input changes end only the
// current audio session; the page run continues with the saved input.
func (p *page) Run(ctx context.Context) {
	// Re-issue the current selection on every activation. If a lookup was
	// cancelled when the page was hidden, the next Run retries it without
	// retaining cross-run goroutines or reading UI state off the event loop.
	p.host.Update(p.Content(), p.requestSelectedCallsign)
	var group errgroup.Group
	group.Go(func() error {
		p.runDecoder(ctx)
		return nil
	})
	group.Go(func() error {
		p.runLookups(ctx)
		return nil
	})
	_ = group.Wait()
}

func (p *page) runDecoder(ctx context.Context) {
	for ctx.Err() == nil {
		p.setStatus("Starting decoder...")
		err := p.runSession(ctx)
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
		if err := p.settingsChanged.Wait(ctx); err != nil {
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
	p.host.Update(p.Content(), func() {
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
	p.showNewQSOEditor(p.Content(), p.selectedCallsign)
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

func (p *page) runSession(ctx context.Context) error {
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
		if err := p.settingsChanged.Wait(sessionCtx); err != nil {
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
	p.host.Update(p.Content(), func() {
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
	p.host.Update(p.Content(), func() {
		p.statusText = status
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
