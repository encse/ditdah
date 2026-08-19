// Package tui implements the Morse decoder terminal user interface.
package tui

import (
	ui "morsemanual/internal/tui"
	"morsemanual/internal/tui/components"
	"morsemanual/internal/tui/keybinding"

	"github.com/rivo/tview"
)

type page struct {
	content tview.Primitive
	output  components.TextView
}

// New creates the page reserved for live Morse decoder output.
func New(host ui.PageHost) ui.Page {
	controls := host.Components()
	output := controls.TextView()
	output.SetStyle(components.TextViewMuted)
	output.SetTextAlign(tview.AlignCenter)
	output.SetText("Decoder output will appear here.")

	content := controls.Flex(tview.FlexRow).
		AddItem(nil, 0, 1, false).
		AddItem(output, 1, 0, false).
		AddItem(nil, 0, 1, false)

	return &page{
		content: content,
		output:  output,
	}
}

func (p *page) ID() string { return "morse-decoder" }

func (p *page) Title() string { return "Morse decoder" }

func (p *page) Content() tview.Primitive { return p.content }

func (p *page) Focusables() []tview.Primitive { return nil }

func (p *page) KeyBindings() []keybinding.Binding { return nil }
