package tui

import (
	"context"
	"errors"
	"strings"
	"testing"

	domain "morsemanual/internal/logbook"

	"github.com/gdamore/tcell/v2"
)

func TestCreateBindingOpensNewQSOEditor(t *testing.T) {
	page, host := newTestPage(t)
	page.qsos = []domain.QSO{{
		ID:              "qso-1",
		StationCallsign: "HA7NCS",
	}}
	page.applyFilter()

	if !page.createBinding().Handle(tcell.NewEventKey(tcell.KeyRune, 'n', 0)) {
		t.Fatal("n was not handled")
	}
	editor, ok := host.modal.(*qsoEditor)
	if !ok {
		t.Fatalf("opened modal = %T, want *qsoEditor", host.modal)
	}
	if editor.qso.ID != "" || editor.qso.Mode != "CW" {
		t.Fatalf("new QSO = %#v", editor.qso)
	}
	if editor.qso.StationCallsign != "HA7NCS" {
		t.Fatalf("station callsign = %q, want selected station", editor.qso.StationCallsign)
	}
	if editor.qso.StartedAt.IsZero() {
		t.Fatal("new QSO start time is zero")
	}
	if editor.title() != " New QSO " {
		t.Fatalf("editor title = %q", editor.title())
	}
}

func TestPageAddsQSOToStoreAndView(t *testing.T) {
	page, _ := newTestPage(t)
	store := &recordingActionStore{}
	page.store = store

	created, err := page.addQSO(domain.QSO{Callsign: "dl1abc"})
	if err != nil {
		t.Fatalf("addQSO() error = %v", err)
	}
	if store.added.Callsign != "dl1abc" {
		t.Fatalf("store received Callsign = %q", store.added.Callsign)
	}
	if created.ID != "new-qso" || len(page.qsos) != 1 {
		t.Fatalf("created QSO = %#v, page QSOs = %#v", created, page.qsos)
	}
	if page.selectedID != "new-qso" {
		t.Fatalf("selected ID = %q, want new-qso", page.selectedID)
	}
}

func TestDeleteBindingConfirmsAndDeletesSelectedQSO(t *testing.T) {
	page, host := newTestPage(t)
	store := &recordingActionStore{}
	page.store = store
	page.qsos = []domain.QSO{
		{ID: "qso-1", Callsign: "DL1ABC"},
		{ID: "qso-2", Callsign: "OE1XYZ"},
	}
	page.applyFilter()

	if !page.deleteBinding().Handle(tcell.NewEventKey(tcell.KeyRune, 'd', 0)) {
		t.Fatal("d was not handled")
	}
	dialog, ok := host.modal.(*confirmDialog)
	if !ok {
		t.Fatalf("opened modal = %T, want *confirmDialog", host.modal)
	}
	if !strings.Contains(dialog.message.Text(), "DL1ABC") {
		t.Fatalf("confirmation = %q, want callsign", dialog.message.Text())
	}
	if len(dialog.KeyBindings()) != 0 {
		t.Fatalf("confirm bindings = %#v, want application-owned Escape", dialog.KeyBindings())
	}

	dialog.confirm.InputHandler()(tcell.NewEventKey(tcell.KeyEnter, 0, 0), nil)

	if store.deleted != "qso-1" {
		t.Fatalf("deleted ID = %q, want qso-1", store.deleted)
	}
	if len(page.qsos) != 1 || page.qsos[0].ID != "qso-2" {
		t.Fatalf("page QSOs = %#v, want qso-2", page.qsos)
	}
	if page.selectedID != "qso-2" {
		t.Fatalf("selected ID = %q, want qso-2", page.selectedID)
	}
	if host.modalHandle == nil || !host.modalHandle.closed {
		t.Fatal("successful delete did not close confirmation")
	}
}

func TestDeleteFailureKeepsConfirmationOpen(t *testing.T) {
	page, host := newTestPage(t)
	page.store = &recordingActionStore{deleteErr: errors.New("disk failed")}
	page.qsos = []domain.QSO{{ID: "qso-1", Callsign: "DL1ABC"}}
	page.applyFilter()
	page.confirmDeleteQSO()
	dialog := host.modal.(*confirmDialog)

	dialog.confirm.InputHandler()(tcell.NewEventKey(tcell.KeyEnter, 0, 0), nil)

	if host.modalHandle.closed {
		t.Fatal("failed delete closed confirmation")
	}
	if !strings.Contains(dialog.detail.Text(), "disk failed") {
		t.Fatalf("confirmation error = %q", dialog.detail.Text())
	}
	if len(page.qsos) != 1 {
		t.Fatal("failed delete changed page model")
	}
}

func TestSyncBindingUploadsPendingQSOsAndRefreshesPage(t *testing.T) {
	page, host := newTestPage(t)
	store := &listingActionStore{}
	syncer := &recordingSynchronizer{}
	page.store = store
	page.syncer = syncer
	page.qsos = []domain.QSO{
		{ID: "qso-1", Callsign: "DL1ABC"},
		{ID: "qso-2", Callsign: "OE1XYZ"},
	}
	page.applyFilter()

	if !page.syncBinding().Handle(tcell.NewEventKey(tcell.KeyRune, 'u', 0)) {
		t.Fatal("u was not handled")
	}
	dialog, ok := host.modal.(*confirmDialog)
	if !ok {
		t.Fatalf("opened modal = %T, want *confirmDialog", host.modal)
	}
	if !strings.Contains(dialog.message.Text(), "2 pending") {
		t.Fatalf("confirmation = %q, want pending count", dialog.message.Text())
	}

	dialog.confirm.InputHandler()(tcell.NewEventKey(tcell.KeyEnter, 0, 0), nil)

	if syncer.calls != 1 {
		t.Fatalf("Sync() calls = %d, want 1", syncer.calls)
	}
	if len(page.qsos) != 0 {
		t.Fatalf("page QSOs = %#v, want refreshed empty list", page.qsos)
	}
	if host.modalHandle == nil || !host.modalHandle.closed {
		t.Fatal("successful sync did not close confirmation")
	}
}

type recordingActionStore struct {
	domain.Store
	added     domain.QSO
	deleted   string
	deleteErr error
}

type listingActionStore struct {
	domain.Store
}

func (s *listingActionStore) List(
	context.Context,
	domain.Filter,
) ([]domain.QSO, error) {
	return nil, nil
}

type recordingSynchronizer struct {
	calls int
}

func (s *recordingSynchronizer) Sync(context.Context) (int, error) {
	s.calls++
	return 2, nil
}

func (s *recordingActionStore) Add(
	_ context.Context,
	qso domain.QSO,
) (domain.QSO, error) {
	s.added = qso
	qso.ID = "new-qso"
	return qso, nil
}

func (s *recordingActionStore) Delete(_ context.Context, id string) error {
	s.deleted = id
	return s.deleteErr
}
