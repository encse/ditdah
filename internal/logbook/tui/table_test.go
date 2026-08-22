package tui

import (
	"fmt"
	"strings"
	"testing"
	"time"

	domain "ditdah/internal/logbook"
	"ditdah/internal/optional"

	"github.com/gdamore/tcell/v2"
)

func TestPageLeavesTableNavigationToControl(t *testing.T) {
	page := page{}
	for _, key := range []rune{'j', 'k', 'h', 'l', 'g', 'G'} {
		event := tcell.NewEventKey(tcell.KeyRune, key, 0)
		for _, binding := range page.KeyBindings() {
			if binding.Handle(event) {
				t.Errorf("page binding handled native table key %q", key)
			}
		}
	}
}

func TestColumnWidths(t *testing.T) {
	var header strings.Builder
	for index, column := range columns {
		if index > 0 {
			header.WriteByte(' ')
		}
		if column.width > 0 {
			fmt.Fprintf(&header, "%-*s", column.width, column.heading)
		} else {
			header.WriteString(column.heading)
		}
	}

	want := "Date        Time   Callsign      Frequency   Mode    QRZ      " +
		"Sent   Received  TX exch     RX exch     Name            QTH"
	if header.String() != want {
		t.Fatalf("header = %q, want %q", header.String(), want)
	}
}

func TestQRZSyncStatus(t *testing.T) {
	if got := qrzSyncStatus(domain.QSO{}); got != "Pending" {
		t.Fatalf("qrzSyncStatus(pending) = %q, want Pending", got)
	}
	qso := domain.QSO{QRZSyncedAt: optional.Some(time.Now())}
	if got := qrzSyncStatus(qso); got != "Synced" {
		t.Fatalf("qrzSyncStatus(synced) = %q, want Synced", got)
	}
}

func TestFormatFrequency(t *testing.T) {
	qso := domain.QSO{FrequencyHz: optional.Some[int64](7_023_500)}
	if got := formatFrequency(qso); got != "7.0235" {
		t.Fatalf("formatFrequency() = %q, want 7.0235", got)
	}
	if got := formatFrequency(domain.QSO{}); got != "" {
		t.Fatalf("formatFrequency(missing) = %q, want empty", got)
	}
}
