package components

import (
	"reflect"
	"testing"

	"morsemanual/internal/tui/keybinding"

	"github.com/gdamore/tcell/v2"
)

func TestInputFieldChangesBackgroundWithFocus(t *testing.T) {
	screen := newTestScreen(t)
	field := newTestFactory().InputField("Callsign", "HA7NCS")
	field.SetFormAttributes(
		10,
		tcell.ColorWhite,
		tcell.ColorBlack,
		tcell.ColorWhite,
		tcell.ColorBlue,
	)
	field.SetRect(2, 2, 30, 1)

	field.Draw(screen)
	assertBackground(t, screen, 12, 2, tcell.ColorBlue)

	field.Focus(nil)
	field.Draw(screen)
	assertBackground(t, screen, 12, 2, testTheme().ActiveFieldBackground)
	assertBackground(t, screen, 31, 2, testTheme().ActiveFieldBackground)

	field.Blur()
	field.Draw(screen)
	assertBackground(t, screen, 12, 2, tcell.ColorBlue)
}

func TestInputFieldPublishesNativeKeyHints(t *testing.T) {
	field := newTestFactory().InputField("Search", "")
	want := []keybinding.Hint{
		{Keys: "Enter", Description: "done"},
		{Keys: "Esc", Description: "cancel"},
		{Keys: "Tab/Shift+Tab", Description: "next/previous"},
	}

	if got := field.KeyHints(); !reflect.DeepEqual(got, want) {
		t.Fatalf("KeyHints() = %#v, want %#v", got, want)
	}
}

func TestInputFieldValue(t *testing.T) {
	field := newTestFactory().InputField("Callsign", "HA7NCS")

	if got := field.Value(); got != "HA7NCS" {
		t.Fatalf("Value() = %q, want %q", got, "HA7NCS")
	}

	field.SetValue("HA5XYZ")
	if got := field.Value(); got != "HA5XYZ" {
		t.Fatalf("Value() = %q, want %q", got, "HA5XYZ")
	}
}

func TestInputFieldCallbacks(t *testing.T) {
	field := newTestFactory().InputField("Search", "")
	var changed string
	var done tcell.Key
	field.SetPlaceholder("callsign...")
	field.SetChangedFunc(func(value string) {
		changed = value
	})
	field.SetDoneFunc(func(key tcell.Key) {
		done = key
	})

	field.SetValue("HA7NCS")
	field.InputHandler()(
		tcell.NewEventKey(tcell.KeyEnter, 0, 0),
		nil,
	)

	if changed != "HA7NCS" {
		t.Fatalf("changed value = %q, want %q", changed, "HA7NCS")
	}
	if done != tcell.KeyEnter {
		t.Fatalf("done key = %v, want %v", done, tcell.KeyEnter)
	}
}
