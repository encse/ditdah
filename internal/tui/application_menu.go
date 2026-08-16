package tui

import (
	"morsemanual/internal/tui/components"

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
	items []components.MenuItem,
) *applicationMenu {
	menuControls := controls.Modal()
	button := menuControls.Menu(applicationMenuLabel, items)
	background := menuControls.TextView()
	return &applicationMenu{
		Flex: tview.NewFlex().
			AddItem(button, applicationMenuButtonWidth, 0, false).
			AddItem(background, 0, 1, false),
		button: button,
	}
}
