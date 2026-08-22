package components

import (
	"testing"

	"ditdah/internal/tui/keybinding"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

func TestMenuBarOrdersMenuAndClickableBindings(t *testing.T) {
	invoked := 0
	bar := newTestFactory().MenuBar(
		"[=]",
		[]MenuItem{{Label: "Settings"}},
		[]keybinding.Binding{
			keybinding.OnKey(tcell.KeyF1, "Logbook", func() { invoked++ }),
			keybinding.OnKey(tcell.KeyF2, "Morse decoder", func() {}),
		},
	)
	screen := newTestScreen(t)
	bar.SetRect(0, 0, bar.Width(), 1)
	bar.Draw(screen)
	if got := bar.(*menuBar).GetItemCount(); got != 3 {
		t.Fatalf("menu bar item count = %d, want hamburger, F1, F2", got)
	}

	assertRune(t, screen, 2, 0, '[')
	assertRune(t, screen, 3, 0, '=')
	assertRune(t, screen, 4, 0, ']')
	assertRune(t, screen, 0, 0, ' ')
	assertRune(t, screen, 1, 0, ' ')
	assertRune(t, screen, 5, 0, ' ')
	assertRune(t, screen, 6, 0, ' ')
	assertRune(t, screen, 7, 0, 'F')
	assertRune(t, screen, 10, 0, 'L')
	assertRune(t, screen, 17, 0, ' ')
	assertRune(t, screen, 18, 0, ' ')
	assertRune(t, screen, 19, 0, 'F')
	consumed, _ := bar.MouseHandler()(
		tview.MouseLeftClick,
		tcell.NewEventMouse(7, 0, tcell.ButtonNone, 0),
		func(tview.Primitive) {},
	)
	if !consumed || invoked != 1 {
		t.Fatalf("F1 click consumed = %v, invocations = %d; want true, 1", consumed, invoked)
	}
}
