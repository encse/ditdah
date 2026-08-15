package tui

import (
	"strings"

	domain "morsemanual/internal/logbook"
	"morsemanual/internal/tui/components"
	"morsemanual/internal/tui/keybinding"

	"github.com/gdamore/tcell/v2"
)

func (p *page) newSearch(controls components.Factory) components.InputField {
	search := controls.InputField(" Search  ", "")
	search.SetPlaceholder("callsign, date, frequency, mode, name, QTH...")
	search.SetChangedFunc(func(string) {
		p.applyFilter()
	})
	search.SetDoneFunc(func(key tcell.Key) {
		switch key {
		case tcell.KeyEnter:
			p.leaveSearch(false)
		case tcell.KeyEscape:
			p.leaveSearch(true)
		}
	})
	return search
}

func (p *page) searchBindings() []keybinding.Binding {
	return []keybinding.Binding{{
		Hint: keybinding.Hint{Keys: "/", Description: "search"},
		Handler: func(event *tcell.EventKey) bool {
			if event.Key() != tcell.KeyRune || event.Rune() != '/' {
				return false
			}
			p.host.SetFocus(p.search)
			return true
		},
	}}
}

func (p *page) leaveSearch(clear bool) {
	if clear {
		p.search.SetValue("")
	}
	p.host.SetFocus(p.table)
}

func (p *page) applyFilter() {
	query := strings.ToLower(strings.TrimSpace(p.search.Value()))

	p.filteredQsos = p.filteredQsos[:0]
	for _, qso := range p.qsos {
		if query == "" || strings.Contains(searchableText(qso), query) {
			p.filteredQsos = append(p.filteredQsos, qso)
		}
	}

	p.refreshView()
}

func searchableText(qso domain.QSO) string {
	values := []string{
		qso.Callsign,
		qso.StationCallsign,
		qso.StartedAt.Local().Format("2006-01-02 15:04"),
		formatFrequency(qso),
		qso.Mode,
		qso.Submode,
		qso.RSTSent,
		qso.RSTReceived,
		qso.ExchangeSent,
		qso.ExchangeReceived,
		qso.Name,
		qso.QTH,
		qso.Notes,
	}
	return strings.ToLower(strings.Join(values, " "))
}
