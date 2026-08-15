package tui

import (
	"strings"
	"testing"
	"time"

	domain "morsemanual/internal/logbook"
	"morsemanual/internal/optional"

	"github.com/gdamore/tcell/v2"
)

func TestSearchBindingFocusesSearch(t *testing.T) {
	page, host := newTestPage(t)
	event := tcell.NewEventKey(tcell.KeyRune, '/', 0)
	handled := false
	for _, binding := range page.KeyBindings() {
		handled = binding.Handle(event) || handled
	}
	if !handled {
		t.Fatal("search binding did not handle /")
	}
	if host.focus != page.search {
		t.Fatalf("focus = %T, want search", host.focus)
	}
}

func TestSearchableTextIncludesLogbookFields(t *testing.T) {
	qso := domain.QSO{
		StationCallsign:  "HA7NCS",
		Callsign:         "DL8ECA/P",
		StartedAt:        time.Date(2026, 8, 13, 16, 17, 0, 0, time.Local),
		FrequencyHz:      optional.Some[int64](7_023_500),
		Mode:             "CW",
		ExchangeReceived: "123",
		Name:             "Flo",
		QTH:              "Remscheid",
		Notes:            "portable",
	}

	text := searchableText(qso)
	for _, expected := range []string{
		"ha7ncs", "dl8eca/p", "7.0235", "cw", "123", "flo",
		"remscheid", "portable",
	} {
		if !strings.Contains(text, expected) {
			t.Errorf("searchableText() = %q, want substring %q", text, expected)
		}
	}
}
