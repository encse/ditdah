package tui

import (
	"testing"

	"ditdah/internal/tui/components"
	"ditdah/internal/tui/keybinding"

	"github.com/gdamore/tcell/v2"
)

func TestApplicationMenuDisplaysModalColoredHamburger(t *testing.T) {
	controls := components.New(components.Dependencies{
		Theme:      nordTheme.components(),
		ModalTheme: nordTheme.modalComponents(),
	})
	menu := newApplicationMenu(controls, []components.MenuItem{{
		Label:       "Exit",
		Hotkey:      'q',
		Description: "quit",
		Action:      func() {},
	}}, []keybinding.Binding{
		keybinding.OnKey(tcell.KeyF1, "Logbook", func() {}),
		keybinding.OnKey(tcell.KeyF2, "Morse decoder", func() {}),
	})
	screen := newLayoutTestScreen(t)
	menu.SetRect(0, 0, menu.Width(), 1)
	menu.Draw(screen)
	assertLayoutRune(t, screen, 2, 0, '[')
	assertLayoutRune(t, screen, 3, 0, '=')
	assertLayoutRune(t, screen, 4, 0, ']')
	assertLayoutRune(t, screen, 7, 0, 'F')
	assertLayoutRune(t, screen, 10, 0, 'L')
	assertLayoutRune(t, screen, 19, 0, 'F')
	_, _, style, _ := screen.GetContent(3, 0)
	menuForeground, background, _ := style.Decompose()
	if want := tcell.NewRGBColor(190, 190, 190); background != want {
		t.Fatalf("menu background = %v, want %v", background, want)
	}
	_, _, style, _ = screen.GetContent(6, 0)
	_, background, _ = style.Decompose()
	if want := tcell.NewRGBColor(190, 190, 190); background != want {
		t.Fatalf("menu edge background = %v, want %v", background, want)
	}
	_, _, style, _ = screen.GetContent(7, 0)
	functionForeground, _, _ := style.Decompose()
	if functionForeground != menuForeground {
		t.Fatalf(
			"function-key color = %v, want menu color %v",
			functionForeground,
			menuForeground,
		)
	}
}
