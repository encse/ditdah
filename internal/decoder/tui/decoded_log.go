package tui

import (
	"strings"

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
	modal.OpenDangerConfirm(
		p.host,
		p.Content(),
		" Clear decoded log ",
		"Clear the decoded log?",
		"",
		"Clear",
		p.clearLog,
	)
}

func (p *page) clearLog() {
	p.decodedText.Reset()
	p.output.Clear()
	p.output.ScrollToEnd()
}
