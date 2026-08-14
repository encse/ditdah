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
	view.Draw(screen)

	assertRune(t, screen, 2, 2, 'H')
	assertForeground(t, screen, 2, 2, testTheme().PrimaryText)
	assertBackground(t, screen, 2, 2, testTheme().Background)
}
