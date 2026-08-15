package components

import (
	"testing"

	"morsemanual/internal/tui/keybinding"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

func TestSelectFieldPublishesHandledBindings(t *testing.T) {
	overlays := &testOverlayHost{}
	field := newTestFactoryWithOverlays(overlays).SelectField(
		"Mode",
		[]string{"CW", "SSB"},
		0,
		6,
		24,
	)
	closedBindings := field.KeyBindings()
	if len(closedBindings) != 1 {
		t.Fatalf("closed binding count = %d, want 1", len(closedBindings))
	}
	if got := keybinding.Hints(closedBindings); len(got) != 0 {
		t.Fatalf("visible closed hints = %#v, want none", got)
	}

	field.Focus(nil)
	field.InputHandler()(tcell.NewEventKey(tcell.KeyEnter, 0, 0), nil)
	popup := overlays.current(t)
	provider, ok := popup.(keybinding.BindingProvider)
	if !ok {
		t.Fatalf("popup type %T does not provide key bindings", popup)
	}
	openBindings := provider.KeyBindings()
	if len(openBindings) != 1 {
		t.Fatalf("popup binding count = %d, want 1", len(openBindings))
	}
	if got := keybinding.Hints(openBindings); len(got) != 0 {
		t.Fatalf("visible popup hints = %#v, want none", got)
	}
}

func TestSelectFieldPopupClosesWhenItLosesFocus(t *testing.T) {
	overlays := &testOverlayHost{}
	field := newTestFactoryWithOverlays(overlays).SelectField(
		"Mode",
		[]string{"CW", "SSB"},
		0,
		6,
		24,
	)
	field.Focus(nil)
	field.InputHandler()(tcell.NewEventKey(tcell.KeyEnter, 0, 0), nil)

	popup := overlays.current(t)
	popup.Blur()

	if overlays.primitive != nil {
		t.Fatal("popup remained open after losing focus")
	}
}

func TestSelectFieldDrawsFullWidthAndBorderedPopup(t *testing.T) {
	screen := newTestScreen(t)
	overlays := &testOverlayHost{}
	field := newTestFactoryWithOverlays(overlays).SelectField(
		"Mode",
		[]string{"CW", "SSB", "DATA"},
		0,
		6,
		24,
	)
	field.SetFormAttributes(
		6,
		tcell.ColorWhite,
		tcell.ColorBlack,
		tcell.ColorWhite,
		tcell.ColorBlue,
	)
	field.SetRect(2, 2, 30, 1)
	field.Focus(nil)
	field.Draw(screen)

	assertRune(t, screen, 30, 2, '▼')
	assertRuneNot(t, screen, 8, 2, tview.Borders.TopLeft)

	field.InputHandler()(tcell.NewEventKey(tcell.KeyEnter, 0, 0), nil)
	field.Draw(screen)
	popup := overlays.current(t)
	popup.SetRect(0, 0, 40, 12)
	popup.Draw(screen)

	assertRune(t, screen, 30, 2, '▲')
	assertRune(t, screen, 8, 3, tview.Borders.TopLeft)
	assertRune(t, screen, 31, 3, tview.Borders.TopRight)
	assertBackground(t, screen, 10, 5, tcell.ColorBlue)
}

func TestSelectFieldUsesThemeBackgroundBehindLabel(t *testing.T) {
	screen := newTestScreen(t)
	field := newTestFactory().SelectField(
		"Mode",
		[]string{"CW", "SSB"},
		0,
		6,
		24,
	)
	field.SetRect(2, 2, 30, 1)
	field.Draw(screen)

	assertBackground(t, screen, 2, 2, testTheme().Background)
	assertBackground(t, screen, 7, 2, testTheme().Background)
	assertBackground(t, screen, 8, 2, testTheme().FieldBackground)
}

func TestSelectFieldSelectsWithMouse(t *testing.T) {
	screen := newTestScreen(t)
	overlays := &testOverlayHost{}
	field := newTestFactoryWithOverlays(overlays).SelectField(
		"Mode",
		[]string{"CW", "SSB", "DATA"},
		0,
		6,
		24,
	)
	field.SetFormAttributes(
		6,
		tcell.ColorWhite,
		tcell.ColorBlack,
		tcell.ColorWhite,
		tcell.ColorBlack,
	)
	field.SetRect(2, 2, 30, 1)
	field.Focus(nil)
	field.Draw(screen)

	handler := field.MouseHandler()
	handler(
		tview.MouseLeftDown,
		tcell.NewEventMouse(10, 2, tcell.Button1, 0),
		func(primitive tview.Primitive) { primitive.Focus(nil) },
	)
	field.Draw(screen)
	popup := overlays.current(t)
	popup.SetRect(0, 0, 40, 12)
	popup.Draw(screen)
	popup.MouseHandler()(
		tview.MouseLeftClick,
		tcell.NewEventMouse(10, 5, tcell.Button1, 0),
		func(primitive tview.Primitive) { primitive.Focus(nil) },
	)

	index, value := field.CurrentOption()
	if index != 1 || value != "SSB" {
		t.Fatalf("CurrentOption() = (%d, %q), want (1, %q)", index, value, "SSB")
	}

	field.Draw(screen)
	assertRune(t, screen, 30, 2, '▼')
}

func TestSelectFieldPopupConsumesOutsideClickAndCloses(t *testing.T) {
	screen := newTestScreen(t)
	overlays := &testOverlayHost{}
	field := newTestFactoryWithOverlays(overlays).SelectField(
		"Mode",
		[]string{"CW", "SSB"},
		0,
		6,
		24,
	)
	field.SetFormAttributes(
		6,
		tcell.ColorWhite,
		tcell.ColorBlack,
		tcell.ColorWhite,
		tcell.ColorBlack,
	)
	field.SetRect(2, 2, 30, 1)
	field.Focus(nil)
	field.InputHandler()(tcell.NewEventKey(tcell.KeyEnter, 0, 0), nil)

	popup := overlays.current(t)
	popup.SetRect(0, 0, 40, 12)
	popup.Draw(screen)
	consumed, capture := popup.MouseHandler()(
		tview.MouseLeftDown,
		tcell.NewEventMouse(0, 0, tcell.Button1, 0),
		func(tview.Primitive) {},
	)

	if !consumed {
		t.Fatal("outside click was not consumed")
	}
	if capture == nil {
		t.Fatal("outside mouse-down did not capture the rest of the click")
	}
	if overlays.primitive != nil {
		t.Fatal("popup remained open after outside click")
	}

	_, capture = popup.MouseHandler()(
		tview.MouseLeftUp,
		tcell.NewEventMouse(0, 0, tcell.ButtonNone, 0),
		func(tview.Primitive) {},
	)
	if capture != nil {
		t.Fatal("popup kept mouse capture after mouse-up")
	}

	field.MouseHandler()(
		tview.MouseLeftDown,
		tcell.NewEventMouse(10, 2, tcell.Button1, 0),
		func(primitive tview.Primitive) { primitive.Focus(nil) },
	)
	if overlays.primitive == nil {
		t.Fatal("popup did not reopen after an outside click")
	}
}

func TestSelectFieldScrollsWhenOptionsDoNotFit(t *testing.T) {
	screen := newTestScreen(t)
	overlays := &testOverlayHost{}
	field := newTestFactoryWithOverlays(overlays).SelectField(
		"Mode",
		[]string{
			"Mode 0", "Mode 1", "Mode 2", "Mode 3",
			"Mode 4", "Mode 5", "Mode 6", "Mode 7",
		},
		0,
		6,
		24,
	)
	field.SetFormAttributes(
		6,
		tcell.ColorWhite,
		tcell.ColorBlack,
		tcell.ColorWhite,
		tcell.ColorBlack,
	)
	field.SetRect(2, 5, 30, 1)
	field.Focus(nil)
	field.InputHandler()(tcell.NewEventKey(tcell.KeyEnter, 0, 0), nil)

	popup := overlays.current(t)
	popup.SetRect(0, 0, 40, 12)
	popup.Draw(screen)

	popup.InputHandler()(tcell.NewEventKey(tcell.KeyPgDn, 0, 0), nil)
	popup.InputHandler()(tcell.NewEventKey(tcell.KeyPgDn, 0, 0), nil)
	popup.Draw(screen)
	popup.InputHandler()(tcell.NewEventKey(tcell.KeyEnter, 0, 0), nil)

	index, value := field.CurrentOption()
	if index != 7 || value != "Mode 7" {
		t.Fatalf("CurrentOption() = (%d, %q), want (7, %q)", index, value, "Mode 7")
	}
}

func TestSelectFieldScrollsWithMouseWheel(t *testing.T) {
	screen := newTestScreen(t)
	overlays := &testOverlayHost{}
	field := newTestFactoryWithOverlays(overlays).SelectField(
		"Mode",
		[]string{
			"Mode 0", "Mode 1", "Mode 2", "Mode 3",
			"Mode 4", "Mode 5", "Mode 6", "Mode 7",
		},
		0,
		6,
		24,
	)
	field.SetFormAttributes(
		6,
		tcell.ColorWhite,
		tcell.ColorBlack,
		tcell.ColorWhite,
		tcell.ColorBlack,
	)
	field.SetRect(2, 8, 30, 1)
	field.Focus(nil)
	field.InputHandler()(tcell.NewEventKey(tcell.KeyEnter, 0, 0), nil)

	popup := overlays.current(t)
	popup.SetRect(0, 0, 40, 12)
	popup.Draw(screen)
	popup.MouseHandler()(
		tview.MouseScrollDown,
		tcell.NewEventMouse(10, 6, tcell.WheelDown, 0),
		func(tview.Primitive) {},
	)
	popup.Draw(screen)
	popup.InputHandler()(tcell.NewEventKey(tcell.KeyEnter, 0, 0), nil)

	index, value := field.CurrentOption()
	if index != 1 || value != "Mode 1" {
		t.Fatalf("CurrentOption() = (%d, %q), want (1, %q)", index, value, "Mode 1")
	}
}
