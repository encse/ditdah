package components

import (
	"testing"

	"github.com/gdamore/tcell/v2"
)

func TestButtonUsesThemeAndInvokesAction(t *testing.T) {
	button := newTestFactory().Button("OK")
	called := false
	button.SetSelectedFunc(func() { called = true })
	button.SetRect(1, 1, 10, 1)
	button.Focus(nil)

	screen := newTestScreen(t)
	button.Draw(screen)
	assertBackground(t, screen, 1, 1, testTheme().ActiveButtonBackground)
	assertForeground(t, screen, 5, 1, testTheme().ActiveButtonText)

	button.InputHandler()(tcell.NewEventKey(tcell.KeyEnter, 0, 0), nil)
	if !called {
		t.Fatal("button action was not invoked")
	}
}

func TestDangerButtonUsesDangerTheme(t *testing.T) {
	button := newTestFactory().DangerButton("Delete")
	button.SetRect(1, 1, 10, 1)
	button.Focus(nil)

	screen := newTestScreen(t)
	button.Draw(screen)
	assertBackground(t, screen, 1, 1, testTheme().ActiveDangerBackground)
	assertForeground(t, screen, 4, 1, testTheme().ActiveButtonText)
}
