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

func TestFooterGivesHintsFullWidthWithoutContext(t *testing.T) {
	footer := newTestFactory().Footer().(*footer)
	if got := footer.GetItemCount(); got != 1 {
		t.Fatalf("item count without context = %d, want 1", got)
	}
	if got := footer.GetItem(0); got != footer.hints {
		t.Fatalf("only item without context = %T, want hints", got)
	}

	footer.SetContext("24 QSOs")
	if got := footer.GetItemCount(); got != 2 {
		t.Fatalf("item count with context = %d, want 2", got)
	}

	footer.SetContext("")
	if got := footer.GetItemCount(); got != 1 {
		t.Fatalf("item count after clearing context = %d, want 1", got)
	}
	if got := footer.GetItem(0); got != footer.hints {
		t.Fatalf("only item after clearing context = %T, want hints", got)
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
