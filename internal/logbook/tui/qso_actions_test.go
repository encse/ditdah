package tui

import (
	"context"
	"errors"
	"strings"
	"testing"

	domain "ditdah/internal/logbook"
	"ditdah/internal/tui/modal"

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
	if host.editors.createdOwner != page || host.editors.createdCallsign != "" {
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
	if host.editors.editedOwner != page || host.editors.editedQSO.ID != "qso-1" {
		t.Fatalf("edit editor call = %p, %#v", host.editors.editedOwner, host.editors.editedQSO)
	}
}

func TestDeleteBindingConfirmsAndDeletesSelectedQSO(t *testing.T) {
	page, host := newTestPage(t)
	backgroundCtx := context.WithValue(context.Background(), "source", "background")
	host.backgroundContext = backgroundCtx
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
	dialog := host.modal
	if dialog == nil {
		t.Fatal("delete did not open confirmation")
	}
	if text := renderedDialogText(t, dialog); !strings.Contains(text, "DL1ABC") {
		t.Fatalf("confirmation = %q, want callsign", text)
	}
	if len(dialog.KeyBindings()) != 0 {
		t.Fatalf("confirm bindings = %#v, want application-owned Escape", dialog.KeyBindings())
	}

	pressDialogButton(t, dialog, 1)

	if store.deleted != "qso-1" {
		t.Fatalf("deleted ID = %q, want qso-1", store.deleted)
	}
	if store.deleteContext != backgroundCtx {
		t.Fatal("delete did not use the Background context")
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

func TestDeleteFailureClosesConfirmationAndOpensError(t *testing.T) {
	page, host := newTestPage(t)
	page.store = &recordingActionStore{deleteErr: errors.New("disk failed")}
	page.qsos = []domain.QSO{{ID: "qso-1", Callsign: "DL1ABC"}}
	page.applyFilter()
	page.confirmDeleteQSO()
	dialog := host.modal
	confirmHandle := host.modalHandle

	pressDialogButton(t, dialog, 1)

	if !confirmHandle.closed {
		t.Fatal("failed delete did not close confirmation")
	}
	errorDialog := host.modal
	if errorDialog == nil || len(errorDialog.Focusables()) != 1 {
		t.Fatalf("opened modal = %T, want message dialog", host.modal)
	}
	if text := renderedDialogText(t, errorDialog); !strings.Contains(text, "disk failed") {
		t.Fatalf("error message = %q", text)
	}
	if len(page.qsos) != 1 {
		t.Fatal("failed delete changed page model")
	}
}

func TestDeleteConfirmationCancelDoesNotDelete(t *testing.T) {
	page, host := newTestPage(t)
	store := &recordingActionStore{}
	page.store = store
	page.qsos = []domain.QSO{{ID: "qso-1", Callsign: "DL1ABC"}}
	page.applyFilter()
	page.confirmDeleteQSO()
	dialog := host.modal
	confirmHandle := host.modalHandle

	pressDialogButton(t, dialog, 0)

	if !confirmHandle.closed {
		t.Fatal("Cancel did not close confirmation")
	}
	if store.deleted != "" {
		t.Fatalf("deleted ID = %q after Cancel", store.deleted)
	}
	if len(page.qsos) != 1 {
		t.Fatal("Cancel changed page model")
	}
}

func TestSyncBindingUploadsPendingQSOsAndRefreshesPage(t *testing.T) {
	page, host := newTestPage(t)
	backgroundCtx := context.WithValue(context.Background(), "source", "background")
	host.backgroundContext = backgroundCtx
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
	dialog := host.modal
	if dialog == nil {
		t.Fatal("sync did not open confirmation")
	}
	if text := renderedDialogText(t, dialog); !strings.Contains(text, "2 pending") {
		t.Fatalf("confirmation = %q, want pending count", text)
	}

	pressDialogButton(t, dialog, 1)

	if syncer.calls != 1 {
		t.Fatalf("Sync() calls = %d, want 1", syncer.calls)
	}
	if syncer.ctx != backgroundCtx {
		t.Fatal("sync did not use the Background context")
	}
	if len(page.qsos) != 0 {
		t.Fatalf("page QSOs = %#v, want refreshed empty list", page.qsos)
	}
	if host.modalHandle == nil || !host.modalHandle.closed {
		t.Fatal("successful sync did not close confirmation")
	}
}

func pressDialogButton(t *testing.T, dialog modal.Dialog, index int) {
	t.Helper()
	focusables := dialog.Focusables()
	if index < 0 || index >= len(focusables) {
		t.Fatalf("dialog button index %d outside %d focusables", index, len(focusables))
	}
	focusables[index].InputHandler()(
		tcell.NewEventKey(tcell.KeyEnter, 0, 0),
		nil,
	)
}

func renderedDialogText(t *testing.T, dialog modal.Dialog) string {
	t.Helper()
	size := dialog.Size()
	screen := newTestScreen(t, size.Width, size.Height)
	dialog.Content().SetRect(0, 0, size.Width, size.Height)
	dialog.Content().Draw(screen)
	var text strings.Builder
	for y := 0; y < size.Height; y++ {
		for x := 0; x < size.Width; x++ {
			character, _, _, _ := screen.GetContent(x, y)
			text.WriteRune(character)
		}
		text.WriteByte('\n')
	}
	return text.String()
}

type recordingActionStore struct {
	domain.Store
	added         domain.QSO
	deleted       string
	deleteContext context.Context
	deleteErr     error
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
	ctx   context.Context
}

func (s *recordingSynchronizer) Sync(ctx context.Context) (int, error) {
	s.calls++
	s.ctx = ctx
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

func (s *recordingActionStore) Delete(ctx context.Context, id string) error {
	s.deleted = id
	s.deleteContext = ctx
	return s.deleteErr
}
