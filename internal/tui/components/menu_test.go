package components

import (
	"morsemanual/internal/tui/keybinding"
	"testing"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

func TestMenuOpensBorderedPopupAndInvokesSelectedItem(t *testing.T) {
	screen := newTestScreen(t)
	overlays := &testOverlayHost{}
	exited := false
	menu := newTestFactoryWithOverlays(overlays).Menu("File", []MenuItem{{
		Label: "Exit",
		Binding: keybinding.OnRune(
			'q', "quit", func() { exited = true },
		),
	}})
	menu.SetRect(5, 1, 10, 1)
	menu.Draw(screen)
	assertRune(t, screen, 8, 1, 'F')

	menu.MouseHandler()(
		tview.MouseLeftDown,
		tcell.NewEventMouse(7, 1, tcell.Button1, 0),
		func(primitive tview.Primitive) { primitive.Focus(nil) },
	)
	popup := overlays.current(t)
	popup.SetRect(0, 0, 40, 12)
	popup.Draw(screen)
	assertRune(t, screen, 5, 2, tview.Borders.TopLeft)
	assertRune(t, screen, 14, 3, 'q')
	assertRune(t, screen, 8, 1, 'F')

	popup.MouseHandler()(
		tview.MouseLeftClick,
		tcell.NewEventMouse(7, 3, tcell.Button1, 0),
		func(tview.Primitive) {},
	)

	if !exited {
		t.Fatal("Exit menu item did not invoke its action")
	}
	if overlays.primitive != nil {
		t.Fatal("menu popup remained open after selection")
	}
}

func TestMenuPopupConsumesOutsideClickAndCloses(t *testing.T) {
	screen := newTestScreen(t)
	overlays := &testOverlayHost{}
	menu := newTestFactoryWithOverlays(overlays).Menu(
		"File",
		[]MenuItem{{Label: "Exit"}},
	)
	menu.SetRect(5, 1, 10, 1)
	menu.InputHandler()(tcell.NewEventKey(tcell.KeyEnter, 0, 0), nil)
	popup := overlays.current(t)
	popup.SetRect(0, 0, 40, 12)
	popup.Draw(screen)

	consumed, _ := popup.MouseHandler()(
		tview.MouseLeftDown,
		tcell.NewEventMouse(0, 0, tcell.Button1, 0),
		func(tview.Primitive) {},
	)

	if !consumed {
		t.Fatal("outside click was not consumed")
	}
	if overlays.primitive != nil {
		t.Fatal("menu popup remained open after outside click")
	}
}
