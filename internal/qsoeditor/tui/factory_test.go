package tui

import (
	"context"
	"testing"

	"morsemanual/internal/callsign"
	domain "morsemanual/internal/logbook"
	"morsemanual/internal/optional"
	"morsemanual/internal/settings"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

func TestFactoryCreatesResolvedQSOForOwner(t *testing.T) {
	_, host := newTestPage(t)
	owner := tview.NewBox()
	store := &factoryStore{}
	var changed domain.QSO
	factory := New(
		host,
		store,
		factorySettings{values: settings.Settings{StationCallsign: "HA7NCS"}},
		factoryLookup{entry: callsign.Entry{Record: optional.Some(callsign.Record{
			Name: optional.Some("Jane Doe"),
			QTH:  optional.Some("Berlin"),
		})}},
		func(qso domain.QSO) { changed = qso },
	)

	factory.Create(owner, " dl1abc ")

	editor, ok := host.modal.(*qsoEditor)
	if !ok {
		t.Fatalf("opened modal = %T, want *qsoEditor", host.modal)
	}
	if host.backgroundOwner != owner || host.modalOwner != owner {
		t.Fatal("draft background work and modal must use the requesting owner")
	}
	if editor.qso.StationCallsign != "HA7NCS" ||
		editor.qso.Callsign != "DL1ABC" || editor.qso.Mode != "CW" ||
		editor.qso.Name != "Jane Doe" || editor.qso.QTH != "Berlin" ||
		editor.qso.StartedAt.IsZero() {
		t.Fatalf("new QSO = %#v", editor.qso)
	}

	editor.ok.InputHandler()(tcell.NewEventKey(tcell.KeyEnter, 0, 0), nil)

	if store.added.Callsign != "DL1ABC" || changed.ID != "new-qso" {
		t.Fatalf("saved QSO = %#v, changed QSO = %#v", store.added, changed)
	}
	if host.modalHandle == nil || !host.modalHandle.closed {
		t.Fatal("successful create did not close the editor")
	}
}

type factoryStore struct {
	domain.Store
	added domain.QSO
}

func (s *factoryStore) Add(
	_ context.Context,
	qso domain.QSO,
) (domain.QSO, error) {
	s.added = qso
	qso.ID = "new-qso"
	return qso, nil
}

type factorySettings struct {
	settings.Store
	values settings.Settings
}

func (s factorySettings) Load(context.Context) (settings.Settings, error) {
	return s.values, nil
}

type factoryLookup struct {
	entry callsign.Entry
}

func (l factoryLookup) Lookup(context.Context, string) (callsign.Entry, error) {
	return l.entry, nil
}
