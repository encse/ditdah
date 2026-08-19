package tui

import (
	"testing"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

func TestCallsignDialogSubmitsInputWithEnter(t *testing.T) {
	var saved string
	dialog := newCallsignDialog(
		newTestHost().Components(),
		"Edit callsign",
		"Save",
		"DL1ABC",
		func(value string) error {
			saved = value
			return nil
		},
	)
	handle := &callsignDialogHandle{}
	dialog.setHandle(handle)
	dialog.input.SetValue("HA7NCS")

	bindings := dialog.input.KeyBindings()
	if len(bindings) != 1 || !bindings[0].Handle(
		tcell.NewEventKey(tcell.KeyEnter, 0, 0),
	) {
		t.Fatal("Enter did not submit the callsign dialog")
	}
	if saved != "HA7NCS" || !handle.closed {
		t.Fatalf("saved = %q, closed = %v", saved, handle.closed)
	}
	if dialog.Size().Height != 6 {
		t.Fatalf("dialog height = %d, want 6", dialog.Size().Height)
	}

	screen := tcell.NewSimulationScreen("UTF-8")
	if err := screen.Init(); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(screen.Fini)
	screen.SetSize(48, 6)
	dialog.Content().SetRect(0, 0, 48, 6)
	dialog.Content().Draw(screen)
	_, _, style, _ := screen.GetContent(3, 1)
	_, background, _ := style.Decompose()
	if want := tcell.NewRGBColor(190, 190, 190); background != want {
		t.Fatalf("modal background = %v, want %v", background, want)
	}
	if character, _, _, _ := screen.GetContent(3, 1); character != ' ' {
		t.Fatalf("row above input starts with %q, want space", character)
	}
	if character, _, _, _ := screen.GetContent(0, 1); character != tview.Borders.Vertical {
		t.Fatalf("outer border contains %q, want one vertical border", character)
	}
	if character, _, _, _ := screen.GetContent(1, 1); character == tview.Borders.Vertical {
		t.Fatalf("dialog has a duplicate inner border at column 1")
	}
	if character, _, _, _ := screen.GetContent(3, 2); character != 'C' {
		t.Fatalf("input row starts with %q, want Callsign", character)
	}
	if character, _, _, _ := screen.GetContent(3, 5); character != tview.Borders.Horizontal {
		t.Fatalf("row below buttons contains %q, want bottom border", character)
	}
}

type callsignDialogHandle struct {
	closed bool
}

func (h *callsignDialogHandle) Close() { h.closed = true }
