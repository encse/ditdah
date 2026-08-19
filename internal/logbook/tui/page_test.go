package tui

import (
	"testing"

	domain "morsemanual/internal/logbook"
)

func TestPageMetadata(t *testing.T) {
	page, _ := newTestPage(t)
	if page.ID() != "logbook" {
		t.Fatalf("ID() = %q, want logbook", page.ID())
	}
	if page.Title() != "Logbook" {
		t.Fatalf("Title() = %q, want Logbook", page.Title())
	}
	if page.Content() == nil {
		t.Fatal("Content() is nil")
	}
	if got := len(page.Focusables()); got != 3 {
		t.Fatalf("Focusables() has %d items, want 3", got)
	}
}

func TestPageRefreshesWholeViewForSelectionChanges(t *testing.T) {
	page, host := newTestPage(t)
	page.qsos = []domain.QSO{
		{ID: "qso-1", Callsign: "HA7NCS"},
		{ID: "qso-2", Callsign: "DL1ABC"},
	}

	page.applyFilter()
	if host.refreshes != 1 || page.selectedID != "qso-1" {
		t.Fatalf("first refresh = %d, selected %q", host.refreshes, page.selectedID)
	}

	page.table.Select(2, 0)
	if host.refreshes != 2 || page.selectedID != "qso-2" {
		t.Fatalf("second refresh = %d, selected %q", host.refreshes, page.selectedID)
	}
}

func TestPageStatusShowsVisibleAndTotalQSOCount(t *testing.T) {
	page, _ := newTestPage(t)
	page.qsos = []domain.QSO{
		{ID: "qso-1"},
		{ID: "qso-2"},
	}
	page.applyFilter()

	if got := page.Status(); got != "(2/2)" {
		t.Fatalf("Status() = %q, want (2/2)", got)
	}
}
