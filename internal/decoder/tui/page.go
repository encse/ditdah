// Package tui implements the Morse decoder terminal user interface.
package tui

import (
	"context"
	"errors"
	"fmt"

	"morsemanual/internal/audio"
	domain "morsemanual/internal/decoder"
	"morsemanual/internal/settings"
	settingspage "morsemanual/internal/settings/tui"
	"morsemanual/internal/trigger"
	ui "morsemanual/internal/tui"
	"morsemanual/internal/tui/components"
	"morsemanual/internal/tui/keybinding"

	"github.com/rivo/tview"
	"golang.org/x/sync/errgroup"
)

var errInputChanged = errors.New("Morse input changed")
var errAudioStopped = errors.New("audio input stopped")

type streamFactory func() (domain.Streaming, error)

type page struct {
	host         ui.PageHost
	source       audio.Source
	settings     settings.Store
	newStream    streamFactory
	content      tview.Primitive
	output       components.TextView
	right        components.TextView
	statusText   string
	inputChanged trigger.Trigger
}

// New creates the page for live Morse decoder output.
func New(
	host ui.PageHost,
	source audio.Source,
	settingsStore settings.Store,
) ui.Page {
	return newPage(host, source, settingsStore, domain.NewStreaming)
}

func newPage(
	host ui.PageHost,
	source audio.Source,
	settingsStore settings.Store,
	newStream streamFactory,
) *page {
	controls := host.Components()
	output := controls.TextView()
	output.SetStyle(components.TextViewPrimary)
	output.SetBorder(" Decoded text ")
	output.SetScrollable(true)
	output.SetWrap(true)
	output.SetWordWrap(true)

	right := controls.TextView()
	right.SetBorder("")

	content := controls.Flex(tview.FlexColumn).
		AddItem(output, 0, 2, true).
		AddItem(right, 0, 1, false)

	return &page{
		host:         host,
		source:       source,
		settings:     settingsStore,
		newStream:    newStream,
		content:      content,
		output:       output,
		right:        right,
		inputChanged: trigger.New(),
		statusText:   "Paused",
	}
}

func (p *page) ID() string { return "morse-decoder" }

func (p *page) Title() string { return "Morse decoder" }

func (p *page) Content() tview.Primitive { return p.content }

func (p *page) Focusables() []tview.Primitive {
	return []tview.Primitive{p.output}
}

func (p *page) KeyBindings() []keybinding.Binding { return nil }

func (p *page) MenuItems(ctx context.Context) []components.MenuItem {
	return []components.MenuItem{{
		Label: "Morse input",
		Binding: keybinding.OnRune('i', "Morse input", func() {
			settingspage.OpenMorseInput(
				ctx,
				p.host,
				p.source,
				p.settings,
				p.inputChanged.Activate,
			)
		}),
	}}
}

func (p *page) Status() string { return p.statusText }

// Run decodes audio while the page is visible. Input changes end only the
// current audio session; the page run continues with the saved input.
func (p *page) Run(ctx context.Context) {
	for ctx.Err() == nil {
		p.setStatus("Starting decoder...")
		err := p.runSession(ctx)
		if ctx.Err() != nil {
			return
		}
		if errors.Is(err, errInputChanged) {
			continue
		}
		if err == nil {
			err = errAudioStopped
		}
		p.setStatus("Error: " + err.Error())
		if err := p.inputChanged.Wait(ctx); err != nil {
			return
		}
	}
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
		return errors.New("select a Morse input from the application menu")
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
		if err := p.inputChanged.Wait(sessionCtx); err != nil {
			return err
		}
		return errInputChanged
	})
	return group.Wait()
}

func (p *page) appendDecoded(ctx context.Context, text string) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	p.host.Update(func() {
		_, _ = p.output.Write([]byte(text))
		p.output.ScrollToEnd()
	})
	return nil
}

func (p *page) setStatus(status string) {
	p.host.Update(func() {
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
