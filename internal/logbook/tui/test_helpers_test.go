package tui

import (
	"testing"

	"morsemanual/internal/tui/components"
	"morsemanual/internal/tui/modal"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

type testHost struct {
	focus       tview.Primitive
	refreshes   int
	controls    components.Factory
	modal       modal.Dialog
	modalHandle *testModalHandle
}

func newTestPage(t *testing.T) (*page, *testHost) {
	t.Helper()
	host := &testHost{controls: components.New(components.Dependencies{
		Theme: testTheme(),
	})}
	return newPage(t.Context(), host, nil), host
}

func (h *testHost) SetFocus(primitive tview.Primitive) {
	h.focus = primitive
	h.Refresh()
}

func (h *testHost) Refresh() {
	h.refreshes++
}

func (h *testHost) Components() components.Factory {
	return h.controls
}

func (h *testHost) OpenModal(dialog modal.Dialog) modal.Handle {
	h.modal = dialog
	h.modalHandle = &testModalHandle{}
	return h.modalHandle
}

type testModalHandle struct {
	closed    bool
	focusNext int
}

func (h *testModalHandle) Close() {
	h.closed = true
}

func (h *testModalHandle) FocusNext() {
	h.focusNext++
}

func testTheme() components.Theme {
	return components.Theme{
		Background:             tcell.ColorBlack,
		PrimaryText:            tcell.ColorWhite,
		SecondaryText:          tcell.ColorSilver,
		MutedText:              tcell.ColorGray,
		Accent:                 tcell.ColorAqua,
		Border:                 tcell.ColorWhite,
		LabelColor:             tcell.ColorWhite,
		FieldTextColor:         tcell.ColorWhite,
		FieldBackground:        tcell.ColorBlue,
		ActiveFieldBackground:  tcell.ColorGreen,
		CursorColor:            tcell.ColorBlack,
		SelectionText:          tcell.ColorBlack,
		SelectionBackground:    tcell.ColorYellow,
		PopupBorder:            tcell.ColorWhite,
		ButtonText:             tcell.ColorWhite,
		ButtonBackground:       tcell.ColorBlue,
		ActiveButtonText:       tcell.ColorBlack,
		ActiveButtonBackground: tcell.ColorAqua,
	}
}

func newTestScreen(t *testing.T, width int, height int) tcell.SimulationScreen {
	t.Helper()
	screen := tcell.NewSimulationScreen("UTF-8")
	if err := screen.Init(); err != nil {
		t.Fatalf("initialize screen: %v", err)
	}
	t.Cleanup(screen.Fini)
	screen.SetSize(width, height)
	return screen
}

func assertRune(t *testing.T, screen tcell.Screen, x int, y int, want rune) {
	t.Helper()
	got, _, _, _ := screen.GetContent(x, y)
	if got != want {
		t.Fatalf("rune at (%d, %d) = %q, want %q", x, y, got, want)
	}
}

func assertBackground(
	t *testing.T,
	screen tcell.Screen,
	x int,
	y int,
	want tcell.Color,
) {
	t.Helper()
	_, _, style, _ := screen.GetContent(x, y)
	_, got, _ := style.Decompose()
	if got != want {
		t.Fatalf("background at (%d, %d) = %v, want %v", x, y, got, want)
	}
}
