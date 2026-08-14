package components

import (
	"testing"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

func newTestFactory() Factory {
	return newTestFactoryWithOverlays(&testOverlayHost{})
}

func newTestFactoryWithOverlays(overlays OverlayHost) Factory {
	return New(Dependencies{
		Theme:    testTheme(),
		Overlays: overlays,
	})
}

type testOverlayHost struct {
	primitive tview.Primitive
}

func (h *testOverlayHost) Push(primitive tview.Primitive) Overlay {
	h.primitive = primitive
	primitive.Focus(nil)
	return &testOverlay{host: h}
}

func (h *testOverlayHost) current(t *testing.T) tview.Primitive {
	t.Helper()
	if h.primitive == nil {
		t.Fatal("no overlay is open")
	}
	return h.primitive
}

type testOverlay struct {
	host *testOverlayHost
}

func (o *testOverlay) Close() {
	if o.host.primitive != nil {
		o.host.primitive.Blur()
		o.host.primitive = nil
	}
}

func newTestScreen(t *testing.T) tcell.SimulationScreen {
	t.Helper()
	screen := tcell.NewSimulationScreen("UTF-8")
	if err := screen.Init(); err != nil {
		t.Fatalf("initialize screen: %v", err)
	}
	t.Cleanup(screen.Fini)
	screen.SetSize(40, 12)
	return screen
}

func testTheme() Theme {
	return Theme{
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

func assertForeground(
	t *testing.T,
	screen tcell.Screen,
	x int,
	y int,
	want tcell.Color,
) {
	t.Helper()
	_, _, style, _ := screen.GetContent(x, y)
	got, _, _ := style.Decompose()
	if got != want {
		t.Fatalf("foreground at (%d, %d) = %v, want %v", x, y, got, want)
	}
}

func assertRune(t *testing.T, screen tcell.Screen, x, y int, want rune) {
	t.Helper()
	got, _, _, _ := screen.GetContent(x, y)
	if got != want {
		t.Fatalf("rune at (%d, %d) = %q, want %q", x, y, got, want)
	}
}

func assertRuneNot(t *testing.T, screen tcell.Screen, x, y int, unwanted rune) {
	t.Helper()
	got, _, _, _ := screen.GetContent(x, y)
	if got == unwanted {
		t.Fatalf("rune at (%d, %d) unexpectedly equals %q", x, y, unwanted)
	}
}
