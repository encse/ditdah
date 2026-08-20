package tui

import (
	"strings"

	"morsemanual/internal/tui/components"
	"morsemanual/internal/tui/keybinding"
	"morsemanual/internal/tui/modal"

	"github.com/rivo/tview"
)

const decodedCallsignRegion = "selected-callsign"

func (p *page) renderDecodedText() {
	wasAtEnd := p.output.AtEnd()
	p.output.SetText(highlightDecodedCallsign(
		p.decodedText.String(),
		p.selectedCallsign,
	))
	p.output.Highlight(decodedCallsignRegion)
	if wasAtEnd {
		p.output.ScrollToEnd()
	}
}

func (p *page) newDecodedMatch(previousLength int) bool {
	selected := strings.ToUpper(p.selectedCallsign)
	if selected == "" {
		return false
	}
	text := p.decodedText.String()
	start := previousLength - len(selected) + 1
	if start < 0 {
		start = 0
	}
	return strings.Contains(strings.ToUpper(text[start:]), selected)
}

func highlightDecodedCallsign(text, selected string) string {
	selected = strings.ToUpper(strings.TrimSpace(selected))
	if selected == "" || text == "" {
		return tview.Escape(text)
	}
	upperText := strings.ToUpper(text)
	var highlighted strings.Builder
	highlighted.Grow(len(text))
	for offset := 0; offset < len(text); {
		relative := strings.Index(upperText[offset:], selected)
		if relative < 0 {
			highlighted.WriteString(tview.Escape(text[offset:]))
			break
		}
		matchStart := offset + relative
		matchEnd := matchStart + len(selected)
		highlighted.WriteString(tview.Escape(text[offset:matchStart]))
		highlighted.WriteString(`["` + decodedCallsignRegion + `"]`)
		highlighted.WriteString(tview.Escape(text[matchStart:matchEnd]))
		highlighted.WriteString(`[""]`)
		offset = matchEnd
	}
	return highlighted.String()
}

func (p *page) confirmClearLog() {
	dialog := newClearLogDialog(p.host.Components(), p.clearLog)
	dialog.setHandle(p.host.OpenModal(dialog))
}

func (p *page) clearLog() {
	p.decodedText.Reset()
	p.output.Clear()
	p.output.ScrollToEnd()
}

type clearLogDialog struct {
	modal.Layout
	clear  components.Button
	cancel components.Button
	action func()
	handle modal.Handle
}

func newClearLogDialog(
	controls components.Factory,
	action func(),
) *clearLogDialog {
	controls = controls.Modal()
	dialog := &clearLogDialog{action: action}
	message := controls.TextView()
	message.SetText("Clear all decoded text?")
	message.SetTextAlign(tview.AlignCenter)
	dialog.clear = controls.DangerButton("Clear")
	dialog.cancel = controls.Button("Cancel")
	dialog.clear.SetSelectedFunc(dialog.submit)
	dialog.cancel.SetSelectedFunc(dialog.close)
	buttons := controls.Flex(tview.FlexColumn).
		AddItem(nil, 0, 1, false).
		AddItem(dialog.cancel, 12, 0, false).
		AddItem(nil, 2, 0, false).
		AddItem(dialog.clear, 12, 0, false).
		AddItem(nil, 0, 1, false)
	body := controls.Flex(tview.FlexRow).
		AddItem(message, 1, 0, false)
	dialog.Layout = modal.NewLayout(
		controls,
		" Clear decoded log ",
		48,
	).Row(body, 1).Actions(buttons)
	return dialog
}

func (d *clearLogDialog) Focusables() []tview.Primitive {
	return []tview.Primitive{d.cancel, d.clear}
}

func (d *clearLogDialog) KeyBindings() []keybinding.Binding { return nil }

func (d *clearLogDialog) setHandle(handle modal.Handle) { d.handle = handle }

func (d *clearLogDialog) submit() {
	if d.action != nil {
		d.action()
	}
	d.close()
}

func (d *clearLogDialog) close() {
	if d.handle != nil {
		d.handle.Close()
	}
}
