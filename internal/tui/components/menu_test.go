package components

import (
	"ditdah/internal/tui/keybinding"
	"testing"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

func TestMenuOpensBorderedPopupAndInvokesSelectedItem(t *testing.T) {
	screen := newTestScreen(t)
	overlays := &testOverlayHost{}
	invocations := 0
	menu := newTestFactoryWithOverlays(overlays).Menu("File", []MenuItem{{
		Label: "Exit",
		Binding: keybinding.OnRune(
			'q', "quit", func() { invocations++ },
		),
	}})
	menu.SetRect(5, 1, 10, 1)
	menu.Draw(screen)
	assertRune(t, screen, 8, 1, 'F')

	menu.MouseHandler()(
		tview.MouseLeftClick,
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
		tview.MouseLeftDown,
		tcell.NewEventMouse(7, 3, tcell.Button1, 0),
		func(tview.Primitive) {},
	)
	popup.MouseHandler()(
		tview.MouseLeftUp,
		tcell.NewEventMouse(7, 3, tcell.ButtonNone, 0),
		func(tview.Primitive) {},
	)
	popup.MouseHandler()(
		tview.MouseLeftClick,
		tcell.NewEventMouse(7, 3, tcell.ButtonNone, 0),
		func(tview.Primitive) {},
	)

	if invocations != 1 {
		t.Fatalf("Exit menu item invocations = %d, want 1", invocations)
	}
	if overlays.primitive != nil {
		t.Fatal("menu popup remained open after selection")
	}
}

func TestMenuPopupPassesOutsideClickThroughAfterClosing(t *testing.T) {
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
		tview.MouseLeftClick,
		tcell.NewEventMouse(0, 0, tcell.Button1, 0),
		func(tview.Primitive) {},
	)

	if consumed {
		t.Fatal("outside click was consumed")
	}
	if overlays.primitive != nil {
		t.Fatal("menu popup remained open after outside click")
	}
}

func TestMenuDoesNotChangeBackgroundWhenFocusedOrOpen(t *testing.T) {
	screen := newTestScreen(t)
	overlays := &testOverlayHost{}
	menu := newTestFactoryWithOverlays(overlays).Menu(
		"File",
		[]MenuItem{{Label: "Exit"}},
	)
	menu.SetRect(0, 0, 6, 1)
	menu.Focus(nil)
	menu.Draw(screen)
	assertBackground(t, screen, 2, 0, testTheme().Background)

	menu.InputHandler()(tcell.NewEventKey(tcell.KeyEnter, 0, 0), nil)
	menu.Draw(screen)
	assertBackground(t, screen, 2, 0, testTheme().Background)
}
