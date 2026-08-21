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
	if host.editors.createdOwner != page.Content() || host.editors.createdCallsign != "" {
		t.Fatalf("create editor call = %p, %q", host.editors.createdOwner, host.editors.createdCallsign)
	}
}

func TestEditBindingOpensSelectedQSOEditor(t *testing.T) {
	page, host := newTestPage(t)
	page.qsos = []domain.QSO{{ID: "qso-1", Callsign: "HA7NCS"}}
	page.applyFilter()

	if !page.editBinding().Handle(tcell.NewEventKey(tcell.KeyEnter, 0, 0)) {
		t.Fatal("Enter was not handled")
	}
	if host.editors.editedOwner != page.Content() || host.editors.editedQSO.ID != "qso-1" {
		t.Fatalf("edit editor call = %p, %#v", host.editors.editedOwner, host.editors.editedQSO)
	}
}

func TestPageAddsCreatedQSOToView(t *testing.T) {
	page, _ := newTestPage(t)
	page.QSOChanged(domain.QSO{ID: "new-qso", Callsign: "DL1ABC"})
	if len(page.qsos) != 1 || page.qsos[0].ID != "new-qso" {
		t.Fatalf("page QSOs = %#v", page.qsos)
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
