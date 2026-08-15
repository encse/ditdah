package components

import (
	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

// Button is an application-styled action button.
type Button interface {
	tview.Primitive
	SetSelectedFunc(handler func())
}

type button struct {
	*tview.Button
}

func newButton(label string, theme Theme, focusChanged func()) Button {
	return newStyledButton(
		label,
		theme,
		theme.ButtonBackground,
		theme.ActiveButtonBackground,
		focusChanged,
	)
}

func newDangerButton(label string, theme Theme, focusChanged func()) Button {
	return newStyledButton(
		label,
		theme,
		theme.DangerButtonBackground,
		theme.ActiveDangerBackground,
		focusChanged,
	)
}

func newStyledButton(
	label string,
	theme Theme,
	background tcell.Color,
	activeBackground tcell.Color,
	focusChanged func(),
) Button {
	view := tview.NewButton(label).
		SetStyle(tcell.StyleDefault.
			Foreground(theme.ButtonText).
			Background(background)).
		SetActivatedStyle(tcell.StyleDefault.
			Foreground(theme.ActiveButtonText).
			Background(activeBackground).
			Bold(true))
	view.SetFocusFunc(func() {
		notify(focusChanged)
	})
	return &button{Button: view}
}

func (b *button) MouseHandler() mouseHandler {
	return keepMouseOwner(b, b.Button.MouseHandler())
}

func (b *button) SetSelectedFunc(handler func()) {
	b.Button.SetSelectedFunc(handler)
}
