package components

import "testing"

func TestModalFactoryUsesModalTheme(t *testing.T) {
	view := newTestFactory().Modal().TextView()
	screen := newTestScreen(t)
	view.SetText("x")
	view.SetRect(0, 0, 20, 3)
	view.Draw(screen)

	assertBackground(t, screen, 1, 1, testModalTheme().Background)
	assertForeground(t, screen, 0, 0, testModalTheme().PrimaryText)
}
