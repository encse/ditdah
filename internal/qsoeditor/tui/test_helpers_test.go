package tui

import (
	"context"
	"testing"

	ui "morsemanual/internal/tui"
	"morsemanual/internal/tui/components"
	"morsemanual/internal/tui/modal"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

type testHost struct {
	controls        components.Factory
	backgroundOwner ui.Owner
	modalOwner      ui.Owner
	modal           modal.Dialog
	modalHandle     *testModalHandle
}

type testOwner struct{ content tview.Primitive }

func (o *testOwner) Content() tview.Primitive { return o.content }

func newTestPage(t *testing.T) (*testOwner, *testHost) {
	t.Helper()
	return &testOwner{content: tview.NewBox()}, &testHost{controls: components.New(components.Dependencies{
		Theme: testTheme(),
	})}
}

func (h *testHost) SetFocus(tview.Primitive) {}
func (h *testHost) Refresh()                 {}

func (h *testHost) Update(_ ui.Owner, update func()) bool {
	if update != nil {
		update()
	}
	return true
}

func (h *testHost) Components() components.Factory { return h.controls }

func (h *testHost) OpenModal(owner ui.Owner, dialog modal.Dialog) modal.Handle {
	h.modalOwner = owner
	h.modal = dialog
	h.modalHandle = &testModalHandle{}
	return h.modalHandle
}

func (h *testHost) Background(owner ui.Owner, work ui.BackgroundWork) bool {
	h.backgroundOwner = owner
	work(context.Background())
	return true
}

type testModalHandle struct{ closed bool }

func (h *testModalHandle) Close() { h.closed = true }

func testTheme() components.Theme {
	return components.Theme{
		Background:             tcell.ColorBlack,
		PrimaryText:            tcell.ColorWhite,
		SecondaryText:          tcell.ColorSilver,
		MutedText:              tcell.ColorGray,
		DangerText:             tcell.ColorRed,
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
		DangerButtonBackground: tcell.ColorMaroon,
		ActiveDangerBackground: tcell.ColorRed,
	}
}

func newTestScreen(t *testing.T, width, height int) tcell.SimulationScreen {
	t.Helper()
	screen := tcell.NewSimulationScreen("UTF-8")
	if err := screen.Init(); err != nil {
		t.Fatalf("initialize screen: %v", err)
	}
	t.Cleanup(screen.Fini)
	screen.SetSize(width, height)
	return screen
}

func assertRune(t *testing.T, screen tcell.Screen, x, y int, want rune) {
	t.Helper()
	got, _, _, _ := screen.GetContent(x, y)
	if got != want {
		t.Fatalf("rune at (%d, %d) = %q, want %q", x, y, got, want)
	}
}

func assertBackground(
	t *testing.T,
	screen tcell.Screen,
	x, y int,
	want tcell.Color,
) {
	t.Helper()
	_, _, style, _ := screen.GetContent(x, y)
	_, got, _ := style.Decompose()
	if got != want {
		t.Fatalf("background at (%d, %d) = %v, want %v", x, y, got, want)
	}
}
