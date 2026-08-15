package tui

import (
	"morsemanual/internal/tui/components"

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

func (t colorTheme) components() components.Theme {
	return components.Theme{
		Background:             t.styles.PrimitiveBackgroundColor,
		PrimaryText:            t.styles.PrimaryTextColor,
		SecondaryText:          t.styles.SecondaryTextColor,
		MutedText:              t.muted,
		Accent:                 t.accent,
		Border:                 t.styles.BorderColor,
		LabelColor:             t.styles.SecondaryTextColor,
		FieldTextColor:         t.styles.PrimaryTextColor,
		FieldBackground:        t.styles.ContrastBackgroundColor,
		ActiveFieldBackground:  t.styles.MoreContrastBackgroundColor,
		CursorColor:            t.accent,
		SelectionText:          t.selectionText,
		SelectionBackground:    t.selectionBackground,
		PopupBorder:            t.styles.BorderColor,
		ButtonText:             t.styles.PrimaryTextColor,
		ButtonBackground:       t.selectionBackground,
		ActiveButtonText:       t.styles.PrimitiveBackgroundColor,
		ActiveButtonBackground: t.accent,
	}
}

func (t colorTheme) modalComponents() components.Theme {
	background := tcell.NewRGBColor(190, 190, 190)
	text := tcell.ColorWhite
	accent := tcell.GetColor("#000080")
	return components.Theme{
		Background:             background,
		PrimaryText:            text,
		SecondaryText:          text,
		MutedText:              tcell.GetColor("#606060"),
		Accent:                 text,
		Border:                 tcell.ColorWhite,
		LabelColor:             text,
		FieldTextColor:         tcell.ColorBlack,
		FieldBackground:        tcell.GetColor("#d8d8d8"),
		ActiveFieldBackground:  tcell.ColorWhite,
		CursorColor:            tcell.GetColor("#88c0d0"),
		SelectionText:          tcell.ColorWhite,
		SelectionBackground:    accent,
		PopupBorder:            accent,
		ButtonText:             tcell.ColorWhite,
		ButtonBackground:       tcell.GetColor("#5e81ac"),
		ActiveButtonText:       tcell.GetColor("#2e3440"),
		ActiveButtonBackground: tcell.GetColor("#88c0d0"),
	}
}
