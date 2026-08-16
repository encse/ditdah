package components

import "testing"

func TestPageStackKeepsItsThemeAndBorderAroundActivePage(t *testing.T) {
	factory := newTestFactory()
	stack := factory.PageStack(" Settings ")
	stack.Add("content", factory.Flex(0), true)
	stack.SetRect(1, 1, 24, 6)

	screen := newTestScreen(t)
	stack.Draw(screen)

	assertBackground(t, screen, 2, 2, testTheme().Background)
	assertForeground(t, screen, 1, 1, testTheme().Border)
}
