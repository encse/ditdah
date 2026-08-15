package tui

import (
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
	})
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

func TestEditorModesKeepsUnknownCurrentMode(t *testing.T) {
	modes, selected := editorModes("FREEDV")
	if selected != 0 || len(modes) == 0 || modes[0] != "FREEDV" {
		t.Fatalf("editorModes(FREEDV) = %#v, %d", modes, selected)
	}
}
