package tui

import (
	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

type colorTheme struct {
	styles              tview.Theme
	accent              tcell.Color
	muted               tcell.Color
	selectionBackground tcell.Color
	selectionText       tcell.Color
}

var nordTheme = func() colorTheme {
	background := tcell.GetColor("#2e3440")
	primary := tcell.GetColor("#eceff4")
	secondary := tcell.GetColor("#d8dee9")
	accent := tcell.GetColor("#88c0d0")

	return colorTheme{
		styles: tview.Theme{
			PrimitiveBackgroundColor:    background,
			ContrastBackgroundColor:     tcell.GetColor("#3b4252"),
			MoreContrastBackgroundColor: tcell.GetColor("#434c5e"),
			BorderColor:                 tcell.GetColor("#4c566a"),
			TitleColor:                  accent,
			GraphicsColor:               accent,
			PrimaryTextColor:            primary,
			SecondaryTextColor:          secondary,
			TertiaryTextColor:           tcell.GetColor("#8fbcbb"),
			InverseTextColor:            background,
			ContrastSecondaryTextColor:  secondary,
		},
		accent:              accent,
		muted:               secondary,
		selectionBackground: tcell.GetColor("#5e81ac"),
		selectionText:       primary,
	}
}()

func colorTag(color tcell.Color) string {
	return "[" + color.Name(true) + "]"
}
