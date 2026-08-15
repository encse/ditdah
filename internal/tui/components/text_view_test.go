package components

import (
	"reflect"
	"testing"

	"morsemanual/internal/tui/keybinding"
)

func TestTextViewPublishesScrollHintsOnlyWhenScrollable(t *testing.T) {
	view := newTestFactory().TextView()
	if got := view.KeyHints(); got != nil {
		t.Fatalf("KeyHints() = %#v before enabling scrolling, want nil", got)
	}

	view.SetScrollable(true)
	want := []keybinding.Hint{
		{Keys: "↑/k ↓/j", Description: "scroll"},
		{Keys: "PgUp/PgDn", Description: "page"},
		{Keys: "Home/End g/G", Description: "first/last"},
	}
	if got := view.KeyHints(); !reflect.DeepEqual(got, want) {
		t.Fatalf("KeyHints() = %#v, want %#v", got, want)
	}

	view.SetScrollable(false)
	if got := view.KeyHints(); got != nil {
		t.Fatalf("KeyHints() = %#v after disabling scrolling, want nil", got)
	}
}
