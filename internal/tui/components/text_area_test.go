package components

import (
	"reflect"
	"testing"

	"morsemanual/internal/tui/keybinding"

	"github.com/gdamore/tcell/v2"
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

func TestTextAreaDoneCallback(t *testing.T) {
	area := newTestFactory().TextArea("Notes", "")
	done := 0
	area.SetBindings(keybinding.OnKey(
		tcell.KeyEscape,
		"cancel",
		func() { done++ },
	))
	area.InputHandler()(tcell.NewEventKey(tcell.KeyEscape, 0, 0), nil)

	if done != 1 {
		t.Fatalf("done calls = %d, want 1", done)
	}
	wantHints := []keybinding.Hint{
		{Keys: "Esc", Description: "cancel"},
	}
	bindings := area.KeyBindings()
	gotHints := make([]keybinding.Hint, len(bindings))
	for index, binding := range bindings {
		gotHints[index] = binding.Hint()
	}
	if !reflect.DeepEqual(gotHints, wantHints) {
		t.Fatalf("KeyBindings() hints = %#v, want %#v", gotHints, wantHints)
	}
	if visible := keybinding.Hints(bindings); len(visible) != 0 {
		t.Fatalf("footer hints = %#v, want none", visible)
	}
}
