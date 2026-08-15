package components

import (
	"testing"
)

func TestTextAreaUsesValueAndFocusBackground(t *testing.T) {
	area := newTestFactory().TextArea("Notes", "first\nsecond")
	area.SetLabelWidth(8)
	area.SetRect(1, 1, 30, 4)
	screen := newTestScreen(t)

	area.Draw(screen)
	if got := area.Value(); got != "first\nsecond" {
		t.Fatalf("Value() = %q, want multiline text", got)
	}
	assertBackground(t, screen, 10, 1, testTheme().FieldBackground)

	area.Focus(nil)
	area.Draw(screen)
	assertBackground(t, screen, 10, 1, testTheme().ActiveFieldBackground)

	area.SetValue("updated")
	if got := area.Value(); got != "updated" {
		t.Fatalf("Value() after SetValue() = %q, want updated", got)
	}
}
