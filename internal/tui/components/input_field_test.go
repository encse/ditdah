package components

import (
	"reflect"
	"testing"

	"ditdah/internal/tui/keybinding"

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

func TestInputFieldPublishesConfiguredBindings(t *testing.T) {
	field := newTestFactory().InputField("Search", "")
	field.SetBindings(
		keybinding.OnKey(
			tcell.KeyEnter,
			"apply",
			func() {},
		),
		keybinding.OnKey(
			tcell.KeyEscape,
			"clear",
			func() {},
		),
	)
	want := []keybinding.Hint{
		{Keys: "Enter", Description: "apply"},
		{Keys: "Esc", Description: "clear"},
	}

	bindings := field.KeyBindings()
	got := make([]keybinding.Hint, len(bindings))
	for index, binding := range bindings {
		got[index] = binding.Hint()
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("KeyBindings() hints = %#v, want %#v", got, want)
	}
	if visible := keybinding.Hints(bindings); len(visible) != 0 {
		t.Fatalf("footer hints = %#v, want none", visible)
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

func TestInputFieldMasksSensitiveValues(t *testing.T) {
	field := newTestFactory().InputField("Password", "secret")
	field.SetMaskCharacter('*')
	field.SetRect(0, 0, 20, 1)

	screen := newTestScreen(t)
	field.Draw(screen)
	assertRune(t, screen, 8, 0, '*')
	if got := field.Value(); got != "secret" {
		t.Fatalf("Value() = %q, want unmasked value", got)
	}
}

func TestInputFieldCallbacks(t *testing.T) {
	field := newTestFactory().InputField("Search", "")
	var changed string
	done := 0
	blurred := 0
	field.SetPlaceholder("callsign...")
	field.SetChangedFunc(func(value string) {
		changed = value
	})
	field.SetBindings(keybinding.OnKey(
		tcell.KeyEnter,
		"done",
		func() { done++ },
	))
	field.SetBlurFunc(func() { blurred++ })

	field.Focus(nil)
	field.SetValue("HA7NCS")
	field.InputHandler()(
		tcell.NewEventKey(tcell.KeyEnter, 0, 0),
		nil,
	)
	field.Blur()

	if changed != "HA7NCS" {
		t.Fatalf("changed value = %q, want %q", changed, "HA7NCS")
	}
	if done != 1 {
		t.Fatalf("done calls = %d, want 1", done)
	}
	if blurred != 1 {
		t.Fatalf("blur calls = %d, want 1", blurred)
	}
}
