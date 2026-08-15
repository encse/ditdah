package components

import (
	"testing"

	"morsemanual/internal/tui/keybinding"
)

func TestFooterKeepsContextAndKeyHintsSeparate(t *testing.T) {
	footer := newTestFactory().Footer().(*footer)
	footer.SetContext("24 QSOs")
	footer.SetKeyHints([]keybinding.Hint{
		{Keys: "↑/↓", Description: "move"},
		{Keys: "q", Description: "quit"},
	})

	if got := footer.context.Text(); got != "24 QSOs" {
		t.Fatalf("context = %q, want %q", got, "24 QSOs")
	}
	wantHints := "[::b]↑/↓[-:-:-] move   [::b]q[-:-:-] quit"
	if got := footer.hints.Text(); got != wantHints {
		t.Fatalf("hints = %q, want %q", got, wantHints)
	}
}

func TestFooterEscapesKeyHintMarkup(t *testing.T) {
	footer := newTestFactory().Footer().(*footer)
	footer.SetKeyHints([]keybinding.Hint{
		{Keys: "[x]", Description: "use [brackets]"},
	})

	want := "[::b][x[][-:-:-] use [brackets[]"
	if got := footer.hints.Text(); got != want {
		t.Fatalf("hints = %q, want %q", got, want)
	}
}
