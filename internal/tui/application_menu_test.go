package tui

import (
	"testing"

	"morsemanual/internal/tui/components"
	"morsemanual/internal/tui/keybinding"

	"github.com/gdamore/tcell/v2"
)

func TestApplicationMenuDisplaysModalColoredHamburger(t *testing.T) {
	controls := components.New(components.Dependencies{
		Theme:      nordTheme.components(),
		ModalTheme: nordTheme.modalComponents(),
	})
	menu := newApplicationMenu(controls, []components.MenuItem{{
		Label:   "Exit",
		Binding: keybinding.OnRune('q', "quit", func() {}),
	}})
	screen := newLayoutTestScreen(t)
	menu.SetRect(0, 0, applicationMenuWidth, 1)
	menu.Draw(screen)
	assertLayoutRune(t, screen, 2, 0, '☰')
	_, _, style, _ := screen.GetContent(2, 0)
	_, background, _ := style.Decompose()
	if want := tcell.NewRGBColor(190, 190, 190); background != want {
		t.Fatalf("menu background = %v, want %v", background, want)
	}
	_, _, style, _ = screen.GetContent(10, 0)
	_, background, _ = style.Decompose()
	if want := tcell.NewRGBColor(190, 190, 190); background != want {
		t.Fatalf("menu spacer background = %v, want %v", background, want)
	}
}
