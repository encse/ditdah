package tui

import (
	"context"
	"testing"
	"time"

	domain "morsemanual/internal/logbook"

	"github.com/gdamore/tcell/v2"
)

func TestEditBindingOpensSelectedQSOEditor(t *testing.T) {
	page, host := newTestPage(t)
	page.qsos = []domain.QSO{{ID: "qso-1", Callsign: "HA7NCS"}}
	page.applyFilter()

	if !page.editBinding().Handle(tcell.NewEventKey(tcell.KeyEnter, 0, 0)) {
		t.Fatal("Enter was not handled")
	}
	editor, ok := host.modal.(*qsoEditor)
	if !ok {
		t.Fatalf("opened modal = %T, want *qsoEditor", host.modal)
	}
	if editor.qso.ID != "qso-1" {
		t.Fatalf("edited QSO = %q, want qso-1", editor.qso.ID)
	}
	if len(editor.Focusables()) != 14 {
		t.Fatalf("focusable count = %d, want 14", len(editor.Focusables()))
	}

	editor.cancel.InputHandler()(tcell.NewEventKey(tcell.KeyEnter, 0, 0), nil)
	if host.modalHandle == nil || !host.modalHandle.closed {
		t.Fatal("Cancel did not close the modal")
	}
}

func TestQSOEditorUsesTwoAlignedColumnsAndFullWidthNotes(t *testing.T) {
	_, host := newTestPage(t)
	editor := newQSOEditor(host.Components(), domain.QSO{
		StationCallsign:  "HA7NCS",
		Callsign:         "DL1ABC",
		StartedAt:        time.Date(2026, 8, 15, 12, 34, 0, 0, time.Local),
		Mode:             "CW",
		RSTSent:          "599",
		RSTReceived:      "579",
		ExchangeSent:     "24",
		ExchangeReceived: "25",
		Name:             "Alice",
		QTH:              "Budapest",
		Notes:            "portable operation",
	}, nil)
	screen := newTestScreen(t, 84, 22)
	editor.Content().SetRect(0, 0, 84, 22)
	editor.Content().Draw(screen)

	assertRune(t, screen, 2, 2, 'M')
	assertRune(t, screen, 43, 2, 'C')
	assertRune(t, screen, 2, 4, 'D')
	assertRune(t, screen, 43, 4, 'F')
	assertRune(t, screen, 2, 6, 'M')
	assertRune(t, screen, 2, 12, 'N')
	assertRune(t, screen, 34, 19, 'O')
	assertRune(t, screen, 46, 19, 'C')
	assertBackground(t, screen, 16, 2, testTheme().FieldBackground)
	assertBackground(t, screen, 57, 2, testTheme().FieldBackground)
	assertBackground(t, screen, 16, 6, testTheme().FieldBackground)
	assertBackground(t, screen, 40, 6, testTheme().FieldBackground)
	assertBackground(t, screen, 2, 15, testTheme().FieldBackground)
	assertBackground(t, screen, 81, 15, testTheme().FieldBackground)
}

func TestQSOEditorShowsCursorInFocusedInput(t *testing.T) {
	_, host := newTestPage(t)
	editor := newQSOEditor(host.Components(), domain.QSO{
		StationCallsign: "HA7NCS",
		StartedAt:       time.Date(2026, 8, 15, 12, 34, 0, 0, time.Local),
		Mode:            "CW",
	}, nil)
	screen := newTestScreen(t, 84, 22)
	editor.Content().SetRect(0, 0, 84, 22)
	editor.stationCallsign.Focus(nil)
	editor.Content().Draw(screen)

	x, y, visible := screen.GetCursor()
	if !visible {
		t.Fatal("focused modal input cursor is hidden")
	}
	if x < 16 || x > 40 || y != 2 {
		t.Fatalf("focused modal input cursor = (%d, %d), want first field", x, y)
	}
	assertBackground(t, screen, 16, 2, testTheme().ActiveFieldBackground)
}

func TestQSOEditorSubmitsEditedValue(t *testing.T) {
	_, host := newTestPage(t)
	originalStartedAt := time.Date(2026, 8, 15, 12, 34, 45, 0, time.Local)
	var submitted domain.QSO
	editor := newQSOEditor(host.Components(), domain.QSO{
		ID:              "qso-1",
		StationCallsign: "HA7NCS",
		Callsign:        "DL1ABC",
		StartedAt:       originalStartedAt,
		Mode:            "CW",
		Submode:         "PCW",
	}, func(qso domain.QSO) (domain.QSO, error) {
		submitted = qso
		return qso, nil
	})
	handle := &testModalHandle{}
	editor.setHandle(handle)
	editor.stationCallsign.SetValue("ha5xyz")
	editor.callsign.SetValue("oe1abc")
	editor.frequency.SetValue("14.04199")
	editor.rstSent.SetValue("599")
	editor.rstReceived.SetValue("579")
	editor.exchangeSent.SetValue("24")
	editor.exchangeReceived.SetValue("25")
	editor.name.SetValue("Alice")
	editor.qth.SetValue("Vienna")
	editor.notes.SetValue("portable")

	editor.ok.InputHandler()(tcell.NewEventKey(tcell.KeyEnter, 0, 0), nil)

	if !handle.closed {
		t.Fatal("successful submit did not close the modal")
	}
	if submitted.ID != "qso-1" ||
		submitted.StationCallsign != "ha5xyz" ||
		submitted.Callsign != "oe1abc" ||
		submitted.Submode != "PCW" ||
		submitted.RSTSent != "599" ||
		submitted.RSTReceived != "579" ||
		submitted.ExchangeSent != "24" ||
		submitted.ExchangeReceived != "25" ||
		submitted.Name != "Alice" ||
		submitted.QTH != "Vienna" ||
		submitted.Notes != "portable" {
		t.Fatalf("submitted QSO = %#v", submitted)
	}
	if !submitted.StartedAt.Equal(originalStartedAt) {
		t.Fatalf("submitted StartedAt = %v, want %v", submitted.StartedAt, originalStartedAt)
	}
	frequency, present := submitted.FrequencyHz.Get()
	if !present || frequency != 14_041_990 {
		t.Fatalf("submitted FrequencyHz = %d, %v", frequency, present)
	}
}

func TestQSOEditorKeepsOpenAndShowsInvalidInput(t *testing.T) {
	_, host := newTestPage(t)
	saveCalls := 0
	editor := newQSOEditor(host.Components(), domain.QSO{
		StartedAt: time.Date(2026, 8, 15, 12, 34, 0, 0, time.Local),
		Mode:      "CW",
	}, func(qso domain.QSO) (domain.QSO, error) {
		saveCalls++
		return qso, nil
	})
	handle := &testModalHandle{}
	editor.setHandle(handle)
	editor.startedAt.SetValue("tomorrow")

	editor.ok.InputHandler()(tcell.NewEventKey(tcell.KeyEnter, 0, 0), nil)

	if handle.closed {
		t.Fatal("invalid submit closed the modal")
	}
	if saveCalls != 0 {
		t.Fatalf("save calls = %d, want 0", saveCalls)
	}
	if editor.message.Text() == "" {
		t.Fatal("invalid submit did not show an error")
	}
}

func TestQSOEditorInputEscapeClosesModal(t *testing.T) {
	_, host := newTestPage(t)
	editor := newQSOEditor(host.Components(), domain.QSO{
		StartedAt: time.Date(2026, 8, 15, 12, 34, 0, 0, time.Local),
		Mode:      "CW",
	}, nil)
	handle := &testModalHandle{}
	editor.setHandle(handle)

	editor.callsign.InputHandler()(
		tcell.NewEventKey(tcell.KeyEscape, 0, 0),
		nil,
	)

	if !handle.closed {
		t.Fatal("focused editor input Escape did not close modal")
	}
}

func TestPageUpdatesStoreAndDisplayedModel(t *testing.T) {
	page, _ := newTestPage(t)
	store := &recordingUpdateStore{}
	page.store = store
	page.qsos = []domain.QSO{{ID: "qso-1", Callsign: "OLD"}}
	page.applyFilter()

	updated, err := page.updateQSO(domain.QSO{
		ID:       "qso-1",
		Callsign: "new",
	})
	if err != nil {
		t.Fatalf("updateQSO() error = %v", err)
	}

	if store.received.Callsign != "new" {
		t.Fatalf("store received Callsign = %q, want new", store.received.Callsign)
	}
	if updated.Callsign != "NEW" {
		t.Fatalf("updated Callsign = %q, want NEW", updated.Callsign)
	}
	if len(page.qsos) != 1 || page.qsos[0].Callsign != "NEW" {
		t.Fatalf("page QSOs = %#v, want returned store model", page.qsos)
	}
}

type recordingUpdateStore struct {
	domain.Store
	received domain.QSO
}

func (s *recordingUpdateStore) Update(
	_ context.Context,
	qso domain.QSO,
) (domain.QSO, error) {
	s.received = qso
	qso.Callsign = "NEW"
	return qso, nil
}

func TestEditorModesKeepsUnknownCurrentMode(t *testing.T) {
	modes, selected := editorModes("FREEDV")
	if selected != 0 || len(modes) == 0 || modes[0] != "FREEDV" {
		t.Fatalf("editorModes(FREEDV) = %#v, %d", modes, selected)
	}
}
