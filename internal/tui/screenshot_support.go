//go:build screenshots

package tui

import (
	"errors"

	"ditdah/internal/tui/keybinding"
	"ditdah/internal/tui/modal"

	"github.com/gdamore/tcell/v2"
)

// DrawScreenshot renders a page and an optional dialog with the real
// application theme and layout. It is available only to the screenshot build.
func DrawScreenshot(
	app Application,
	page Page,
	dialog modal.Dialog,
	screen tcell.SimulationScreen,
) error {
	concrete, ok := app.(*application)
	if !ok {
		return errors.New("screenshot application has an unexpected type")
	}

	width, height := screen.Size()
	concrete.engine.SetScreen(screen)
	screen.SetSize(width, height)
	concrete.initializeRuntime(func() {})
	concrete.showPage(page)
	if dialog != nil {
		size := dialog.Size()
		opened := &openedModal{
			dialog:   dialog,
			bindings: dialog.KeyBindings(),
			layer: newModalLayer(
				dialog.Content(),
				size.Width,
				size.Height,
				concrete.theme.styles.BorderColor,
				concrete.theme.styles.PrimitiveBackgroundColor,
			),
		}
		opened.closeBinding = keybinding.OnKey(
			tcell.KeyEscape,
			"close",
			func() {},
		)
		concrete.showModal(opened)
	}
	concrete.Refresh()

	concrete.root.SetRect(0, 0, width, height)
	concrete.root.Draw(screen)
	screen.Show()
	return nil
}
