package tui

import (
	"fmt"
	"strings"
	"testing"
	"time"

	"morsemanual/internal/logbook"
	"morsemanual/internal/optional"

	"github.com/gdamore/tcell/v2"
)

func TestLogbookLeavesTableNavigationKeysToFocusedControl(t *testing.T) {
	view := logbookView{}
	for _, key := range []rune{'j', 'k', 'h', 'l', 'g', 'G'} {
		event := tcell.NewEventKey(tcell.KeyRune, key, 0)
		if got := view.captureKey(event); got != event {
			t.Errorf("captureKey(%q) = %v, want original event", key, got)
		}
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
