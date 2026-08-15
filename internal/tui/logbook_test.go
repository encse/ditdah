package tui

import (
	"fmt"
	"strings"
	"testing"
	"time"

	"morsemanual/internal/logbook"
	"morsemanual/internal/optional"
	"morsemanual/internal/tui/components"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

func TestLogbookPageMetadataAndSearchBinding(t *testing.T) {
	controls := components.New(components.Dependencies{
		Theme: nordTheme.components(),
	})
	host := &testPageHost{
		controls: controls,
		theme:    nordTheme,
	}
	page := newLogbookPage(
		t.Context(),
		host,
		nil,
	)

	if got := page.ID(); got != "logbook" {
		t.Fatalf("ID() = %q, want %q", got, "logbook")
	}
	if got := page.Title(); got != "Logbook" {
		t.Fatalf("Title() = %q, want %q", got, "Logbook")
	}
	if page.Content() == nil {
		t.Fatal("Content() is nil")
	}

	event := tcell.NewEventKey(tcell.KeyRune, '/', 0)
	bindings := page.KeyBindings()
	if len(bindings) != 1 || !bindings[0].Handle(event) {
		t.Fatal("search binding did not handle /")
	}
	if got := host.focus; got != page.search {
		t.Fatalf("focus = %T, want logbook search", got)
	}
}

type testPageHost struct {
	focus     tview.Primitive
	refreshes int
	controls  components.Factory
	theme     colorTheme
}

func (h *testPageHost) SetFocus(primitive tview.Primitive) {
	h.focus = primitive
	h.Refresh()
}

func (h *testPageHost) Refresh() {
	h.refreshes++
}

func (h *testPageHost) Components() components.Factory {
	return h.controls
}

func (h *testPageHost) Theme() colorTheme {
	return h.theme
}

func TestLogbookLeavesTableNavigationKeysToFocusedControl(t *testing.T) {
	page := logbookPage{}
	for _, key := range []rune{'j', 'k', 'h', 'l', 'g', 'G'} {
		event := tcell.NewEventKey(tcell.KeyRune, key, 0)
		for _, binding := range page.KeyBindings() {
			if binding.Handle(event) {
				t.Errorf("page binding handled native table key %q", key)
			}
		}
	}
}

func TestLogbookRefreshesWholeViewForFilterAndSelectionChanges(t *testing.T) {
	host := &testPageHost{
		controls: components.New(components.Dependencies{
			Theme: nordTheme.components(),
		}),
		theme: nordTheme,
	}
	page := newLogbookPage(t.Context(), host, nil)
	page.qsos = []logbook.QSO{
		{ID: "qso-1", Callsign: "HA7NCS"},
		{ID: "qso-2", Callsign: "DL1ABC"},
	}

	page.applyFilter()

	if host.refreshes != 1 {
		t.Fatalf("full refresh count = %d, want 1", host.refreshes)
	}
	if details := page.details.Text(); !strings.Contains(details, "HA7NCS") {
		t.Fatalf("details = %q, want selected QSO", details)
	}

	page.table.Select(2, 0)
	if host.refreshes != 2 {
		t.Fatalf("refresh count after selection = %d, want 2", host.refreshes)
	}
	if details := page.details.Text(); !strings.Contains(details, "DL1ABC") {
		t.Fatalf("details after selection = %q, want selected QSO", details)
	}
}

func TestLogbookColumnWidths(t *testing.T) {
	var header strings.Builder
	for index, column := range logbookColumns {
		if index > 0 {
			header.WriteByte(' ')
		}
		if column.width > 0 {
			fmt.Fprintf(&header, "%-*s", column.width, column.heading)
		} else {
			header.WriteString(column.heading)
		}
	}

	want := "Date        Time   Callsign      Frequency   Mode    Sent   " +
		"Received  TX exch     RX exch     Name            QTH"
	if got := header.String(); got != want {
		t.Fatalf("header = %q, want %q", got, want)
	}
}

func TestSearchableTextIncludesLogbookFields(t *testing.T) {
	qso := logbook.QSO{
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

func TestFormatFrequency(t *testing.T) {
	qso := logbook.QSO{FrequencyHz: optional.Some[int64](7_023_500)}
	if got := formatFrequency(qso); got != "7.0235" {
		t.Fatalf("formatFrequency() = %q, want %q", got, "7.0235")
	}

	if got := formatFrequency(logbook.QSO{}); got != "" {
		t.Fatalf("formatFrequency(missing) = %q, want empty", got)
	}
}
