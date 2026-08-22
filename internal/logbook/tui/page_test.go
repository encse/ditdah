package tui

import (
	"context"
	"sync"
	"testing"
	"time"

	domain "morsemanual/internal/logbook"
	"morsemanual/internal/syncutil"
)

func TestPageLoadsLogbookAndRefreshesItAfterChanges(t *testing.T) {
	page, host := newTestPage(t)
	host.updates = make(chan struct{}, 1)
	store := newRunListingStore([]domain.QSO{{ID: "qso-1", Callsign: "DL1ABC"}})
	page.store = store
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})
	go func() {
		page.Run(ctx)
		close(done)
	}()

	select {
	case <-host.updates:
	case <-time.After(time.Second):
		t.Fatal("Run did not load the logbook")
	}
	if store.ctx != ctx {
		t.Fatal("List did not use the Run context")
	}
	if len(page.qsos) != 1 || page.qsos[0].ID != "qso-1" {
		t.Fatalf("loaded QSOs = %#v", page.qsos)
	}

	store.setQSOs([]domain.QSO{{ID: "qso-2", Callsign: "OE1XYZ"}})
	select {
	case <-host.updates:
	case <-time.After(time.Second):
		t.Fatal("Run did not refresh the changed logbook")
	}
	if len(page.qsos) != 1 || page.qsos[0].ID != "qso-2" {
		t.Fatalf("refreshed QSOs = %#v", page.qsos)
	}

	cancel()
	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("Run did not stop after context cancellation")
	}
}

type runListingStore struct {
	domain.Store
	mu      sync.Mutex
	ctx     context.Context
	qsos    []domain.QSO
	changes syncutil.Broadcaster
}

func newRunListingStore(qsos []domain.QSO) *runListingStore {
	return &runListingStore{
		qsos:    qsos,
		changes: syncutil.NewBroadcaster(),
	}
}

func (s *runListingStore) List(
	ctx context.Context,
	_ domain.Filter,
) ([]domain.QSO, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.ctx = ctx
	return append([]domain.QSO(nil), s.qsos...), nil
}

func (s *runListingStore) Subscribe() syncutil.Subscription {
	return s.changes.Subscribe()
}

func (s *runListingStore) setQSOs(qsos []domain.QSO) {
	s.mu.Lock()
	s.qsos = qsos
	s.mu.Unlock()
	s.changes.Activate()
}

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
