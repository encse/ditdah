package logbook

import (
	"testing"

	"morsemanual/internal/tui/components"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

type testHost struct {
	focus     tview.Primitive
	refreshes int
	controls  components.Factory
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

func testTheme() components.Theme {
	return components.Theme{
		Background:            tcell.ColorBlack,
		PrimaryText:           tcell.ColorWhite,
		SecondaryText:         tcell.ColorSilver,
		MutedText:             tcell.ColorGray,
		Accent:                tcell.ColorAqua,
		Border:                tcell.ColorWhite,
		LabelColor:            tcell.ColorWhite,
		FieldTextColor:        tcell.ColorWhite,
		FieldBackground:       tcell.ColorBlue,
		ActiveFieldBackground: tcell.ColorGreen,
		SelectionText:         tcell.ColorBlack,
		SelectionBackground:   tcell.ColorYellow,
		PopupBorder:           tcell.ColorWhite,
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
