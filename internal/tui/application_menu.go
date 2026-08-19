package tui

import (
	"morsemanual/internal/tui/components"
	"morsemanual/internal/tui/keybinding"
)

const (
	// Keep this ASCII-only. Some terminals render the Unicode hamburger as two
	// cells while tcell measures it as one, shifting everything after it.
	applicationMenuLabel = "[=]"
)

type applicationMenu struct {
	components.MenuBar
	button components.Menu
}

func newApplicationMenu(
	controls components.Factory,
	items []components.MenuItem,
	bindings []keybinding.Binding,
) *applicationMenu {
	menuControls := controls.Modal()
	bar := menuControls.MenuBar(applicationMenuLabel, items, bindings)
	return &applicationMenu{
		MenuBar: bar,
		button:  bar.Button(),
	}
}
