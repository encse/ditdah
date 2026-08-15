package logbook

import (
	"testing"

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
}
