package tui

import (
	"morsemanual/internal/tui/components"
	"morsemanual/internal/tui/keybinding"

	"github.com/rivo/tview"
)

const (
	applicationMenuLabel       = "☰"
	applicationMenuButtonWidth = 5
	applicationMenuWidth       = 16
)

type applicationMenu struct {
	*tview.Flex
	button components.Menu
}

func newApplicationMenu(
	controls components.Factory,
	exit keybinding.Binding,
) *applicationMenu {
	menuControls := controls.Modal()
	button := menuControls.Menu(applicationMenuLabel, []components.MenuItem{{
		Label:   "Exit",
		Binding: exit,
	}})
	background := menuControls.TextView()
	return &applicationMenu{
		Flex: tview.NewFlex().
			AddItem(button, applicationMenuButtonWidth, 0, false).
			AddItem(background, 0, 1, false),
		button: button,
	}
}
