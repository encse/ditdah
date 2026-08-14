package components

import (
	"testing"

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
