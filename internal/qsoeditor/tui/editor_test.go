package tui

import (
	"context"
	"testing"
	"time"

	"ditdah/internal/callsign"
	domain "ditdah/internal/logbook"
	"ditdah/internal/optional"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

func TestQSOEditorUsesTwoAlignedColumnsAndFullWidthNotes(t *testing.T) {
	_, host := newTestPage(t)
	editor := newQSOEditor(host, domain.QSO{
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
	}, nil, nil)
	screen := newTestScreen(t, 84, 22)
	editor.Content().SetRect(0, 0, 84, 22)
	editor.Content().Draw(screen)
	if label := editor.frequency.GetLabel(); label != "Frequency (MHz)" {
		t.Fatalf("frequency label = %q, want Frequency (MHz)", label)
	}

	assertRune(t, screen, 3, 2, 'M')
	assertRune(t, screen, 43, 2, 'C')
	assertRune(t, screen, 3, 4, 'D')
	assertRune(t, screen, 43, 4, 'F')
	assertRune(t, screen, 3, 6, 'M')
	assertRune(t, screen, 3, 12, 'N')
	assertRune(t, screen, 34, 20, 'O')
	assertRune(t, screen, 46, 20, 'C')
	assertRune(t, screen, 3, 21, tview.Borders.Horizontal)
	assertBackground(t, screen, 19, 2, testTheme().FieldBackground)
	assertBackground(t, screen, 59, 2, testTheme().FieldBackground)
	assertBackground(t, screen, 19, 6, testTheme().FieldBackground)
	assertBackground(t, screen, 40, 6, testTheme().FieldBackground)
	assertBackground(t, screen, 3, 15, testTheme().FieldBackground)
	assertBackground(t, screen, 80, 15, testTheme().FieldBackground)
}

func TestQSOEditorShowsCursorInFocusedInput(t *testing.T) {
	_, host := newTestPage(t)
	editor := newQSOEditor(host, domain.QSO{
		StationCallsign: "HA7NCS",
		StartedAt:       time.Date(2026, 8, 15, 12, 34, 0, 0, time.Local),
		Mode:            "CW",
	}, nil, nil)
	screen := newTestScreen(t, 84, 22)
	editor.Content().SetRect(0, 0, 84, 22)
	editor.stationCallsign.Focus(nil)
	editor.Content().Draw(screen)

	x, y, visible := screen.GetCursor()
	if !visible {
		t.Fatal("focused modal input cursor is hidden")
	}
	if x < 19 || x > 40 || y != 2 {
		t.Fatalf("focused modal input cursor = (%d, %d), want first field", x, y)
	}
	assertBackground(t, screen, 19, 2, testTheme().ActiveFieldBackground)
}

func TestQSOEditorSubmitsEditedValue(t *testing.T) {
	_, host := newTestPage(t)
	originalStartedAt := time.Date(2026, 8, 15, 12, 34, 45, 0, time.Local)
	var submitted domain.QSO
	editor := newQSOEditor(host, domain.QSO{
		ID:              "qso-1",
		StationCallsign: "HA7NCS",
		Callsign:        "DL1ABC",
		StartedAt:       originalStartedAt,
		Mode:            "CW",
		Submode:         "PCW",
	}, func(_ context.Context, qso domain.QSO) (domain.QSO, error) {
		submitted = qso
		return qso, nil
	}, nil)
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
	editor := newQSOEditor(host, domain.QSO{
		StartedAt: time.Date(2026, 8, 15, 12, 34, 0, 0, time.Local),
		Mode:      "CW",
	}, func(_ context.Context, qso domain.QSO) (domain.QSO, error) {
		saveCalls++
		return qso, nil
	}, nil)
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

func TestQSOEditorCallsignInputOnlyOwnsLookupEnter(t *testing.T) {
	_, host := newTestPage(t)
	editor := newQSOEditor(host, domain.QSO{
		StartedAt: time.Date(2026, 8, 15, 12, 34, 0, 0, time.Local),
		Mode:      "CW",
	}, nil, nil)
	handle := &testModalHandle{}
	editor.setHandle(handle)
	bindings := editor.callsign.KeyBindings()
	if len(bindings) != 1 || bindings[0].Hint().Keys != "Enter" {
		t.Fatalf("input bindings = %#v, want lookup Enter", bindings)
	}

	editor.callsign.InputHandler()(
		tcell.NewEventKey(tcell.KeyEscape, 0, 0),
		nil,
	)

	if handle.closed {
		t.Fatal("focused editor input handled application-owned modal Escape")
	}
}

func TestQSOEditorRefreshesCallsignDataOnEnterAndBlur(t *testing.T) {
	_, host := newTestPage(t)
	lookup := &editorCallsignLookup{entries: map[string]callsign.Entry{
		"OE1ABC": {
			Record: optional.Some(callsign.Record{
				Name: optional.Some("Alice"),
				QTH:  optional.Some("Vienna"),
			}),
		},
		"DL1XYZ": {
			Record: optional.Some(callsign.Record{
				Nickname: optional.Some("Bob"),
				QTH:      optional.Some("Berlin"),
			}),
		},
	}}
	editor := newQSOEditor(host, domain.QSO{
		Callsign:  "OLD",
		StartedAt: time.Date(2026, 8, 15, 12, 34, 0, 0, time.Local),
		Mode:      "CW",
		Name:      "Old name",
		QTH:       "Old QTH",
	}, nil, lookup)

	editor.callsign.SetValue(" oe1abc ")
	editor.callsign.InputHandler()(
		tcell.NewEventKey(tcell.KeyEnter, 0, 0),
		nil,
	)
	if editor.name.Value() != "Alice" || editor.qth.Value() != "Vienna" {
		t.Fatalf("Enter lookup fields = %q, %q", editor.name.Value(), editor.qth.Value())
	}
	if len(lookup.calls) != 1 || lookup.calls[0] != "OE1ABC" {
		t.Fatalf("Enter lookup calls = %#v, want OE1ABC", lookup.calls)
	}

	editor.callsign.InputHandler()(
		tcell.NewEventKey(tcell.KeyEnter, 0, 0),
		nil,
	)
	if len(lookup.calls) != 1 {
		t.Fatalf("unchanged callsign lookup calls = %#v", lookup.calls)
	}

	editor.callsign.SetValue("dl1xyz")
	editor.callsign.Focus(nil)
	editor.callsign.Blur()
	if editor.name.Value() != "Bob" || editor.qth.Value() != "Berlin" {
		t.Fatalf("blur lookup fields = %q, %q", editor.name.Value(), editor.qth.Value())
	}
	if len(lookup.calls) != 2 || lookup.calls[1] != "DL1XYZ" {
		t.Fatalf("blur lookup calls = %#v, want DL1XYZ", lookup.calls)
	}
}

func TestQSOEditorInputEnterDoesNotMoveFocus(t *testing.T) {
	_, host := newTestPage(t)
	editor := newQSOEditor(host, domain.QSO{
		StartedAt: time.Date(2026, 8, 15, 12, 34, 0, 0, time.Local),
		Mode:      "CW",
	}, nil, nil)
	handle := &testModalHandle{}
	editor.setHandle(handle)

	editor.callsign.InputHandler()(
		tcell.NewEventKey(tcell.KeyEnter, 0, 0),
		nil,
	)

	if handle.closed {
		t.Fatal("input Enter closed modal")
	}
}

func TestQSOEditorLeavesModalBindingsToApplication(t *testing.T) {
	_, host := newTestPage(t)
	editor := newQSOEditor(host, domain.QSO{
		StartedAt: time.Date(2026, 8, 15, 12, 34, 0, 0, time.Local),
		Mode:      "CW",
	}, nil, nil)
	bindings := editor.KeyBindings()

	if len(bindings) != 0 {
		t.Fatalf("editor bindings = %#v, want none", bindings)
	}
}

type editorCallsignLookup struct {
	entries map[string]callsign.Entry
	calls   []string
}

func (s *editorCallsignLookup) Lookup(
	_ context.Context,
	value string,
) (callsign.Entry, error) {
	s.calls = append(s.calls, value)
	return s.entries[value], nil
}

func TestEditorModesKeepsUnknownCurrentMode(t *testing.T) {
	modes, selected := editorModes("FREEDV")
	if selected != 0 || len(modes) == 0 || modes[0] != "FREEDV" {
		t.Fatalf("editorModes(FREEDV) = %#v, %d", modes, selected)
	}
}
