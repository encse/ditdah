package tui

import (
	"testing"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

func TestModalLayerCentersContentAndDimsUnderlyingScreen(t *testing.T) {
	screen := newModalLayerTestScreen(t, 40, 12)
	underlying := tcell.StyleDefault.
		Foreground(tcell.ColorWhite).
		Background(tcell.ColorRed)
	screen.SetContent(0, 0, 'x', nil, underlying)

	content := tview.NewBox()
	layer := newModalLayer(
		content,
		20,
		6,
		tcell.ColorGray,
		tcell.ColorBlack,
	)
	layer.SetRect(0, 0, 40, 12)
	layer.Draw(screen)

	x, y, width, height := content.GetRect()
	if x != 10 || y != 3 || width != 20 || height != 6 {
		t.Fatalf(
			"content rect = (%d, %d, %d, %d), want (10, 3, 20, 6)",
			x, y, width, height,
		)
	}
	character, _, style, _ := screen.GetContent(0, 0)
	foreground, background, _ := style.Decompose()
	if character != 'x' || foreground != tcell.ColorGray || background != tcell.ColorBlack {
		t.Fatalf(
			"dimmed cell = (%q, %v, %v), want ('x', gray, black)",
			character, foreground, background,
		)
	}
}

func TestModalLayerConsumesMouseOutsideDialog(t *testing.T) {
	content := tview.NewBox()
	content.SetRect(10, 3, 20, 6)
	layer := newModalLayer(
		content,
		20,
		6,
		tcell.ColorGray,
		tcell.ColorBlack,
	)
	layer.SetRect(0, 0, 40, 12)

	consumed, capture := layer.MouseHandler()(
		tview.MouseLeftDown,
		tcell.NewEventMouse(1, 1, tcell.Button1, 0),
		func(tview.Primitive) {
			t.Fatal("outside click changed focus")
		},
	)
	if !consumed || capture != nil {
		t.Fatalf("outside click = (%t, %T), want consumed without capture", consumed, capture)
	}
}

func newModalLayerTestScreen(
	t *testing.T,
	width int,
	height int,
) tcell.SimulationScreen {
	t.Helper()
	screen := tcell.NewSimulationScreen("UTF-8")
	if err := screen.Init(); err != nil {
		t.Fatalf("initialize screen: %v", err)
	}
	t.Cleanup(screen.Fini)
	screen.SetSize(width, height)
	return screen
}
