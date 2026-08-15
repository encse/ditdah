package components

import (
	"fmt"
	"testing"
)

func TestTextViewUsesThemeAndImplementsWriter(t *testing.T) {
	screen := newTestScreen(t)
	view := newTestFactory().TextView()
	view.SetRect(2, 2, 20, 1)

	if _, err := fmt.Fprint(view, "Hello"); err != nil {
		t.Fatalf("write text: %v", err)
	}
	if got := view.Text(); got != "Hello" {
		t.Fatalf("Text() = %q, want %q", got, "Hello")
	}
	view.Draw(screen)

	assertRune(t, screen, 2, 2, 'H')
	assertForeground(t, screen, 2, 2, testTheme().PrimaryText)
	assertBackground(t, screen, 2, 2, testTheme().Background)
}

func TestTextViewAppliesSemanticStyle(t *testing.T) {
	screen := newTestScreen(t)
	view := newTestFactory().TextView()
	view.SetText("Status")
	view.SetStyle(TextViewMuted)
	view.SetRect(0, 0, 10, 1)
	view.Draw(screen)

	assertForeground(t, screen, 0, 0, testTheme().MutedText)
}
