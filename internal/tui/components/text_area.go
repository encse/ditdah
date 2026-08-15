package components

import (
	"morsemanual/internal/tui/keybinding"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

// TextArea is an application-styled multiline text input.
type TextArea interface {
	tview.FormItem
	keybinding.HintProvider
	keybinding.ParentBindingBlocker
	Value() string
	SetValue(value string)
	SetLabelWidth(width int)
	SetDoneFunc(handler func(key tcell.Key))
}

type textArea struct {
	*tview.TextArea
	theme Theme
}

func (a *textArea) Draw(screen tcell.Screen) {
	a.TextArea.Draw(screen)
	if a.HasFocus() {
		screen.SetCursorStyle(tcell.CursorStyleDefault, a.theme.CursorColor)
	}
}

func newTextArea(
	label string,
	value string,
	theme Theme,
	focusChanged func(),
) TextArea {
	area := &textArea{
		TextArea: tview.NewTextArea().
			SetLabel(label).
			SetText(value, false).
			SetLabelStyle(tcell.StyleDefault.Foreground(theme.LabelColor)).
			SetSelectedStyle(tcell.StyleDefault.
				Foreground(theme.SelectionText).
				Background(theme.SelectionBackground)).
			SetPlaceholderStyle(tcell.StyleDefault.
				Foreground(theme.MutedText).
				Background(theme.FieldBackground)),
		theme: theme,
	}
	area.SetFocusFunc(func() {
		area.showFocused()
		notify(focusChanged)
	})
	area.SetBlurFunc(area.showIdle)
	area.showIdle()
	return area
}

func (a *textArea) BlocksParentBindings() bool {
	return true
}

func (a *textArea) KeyHints() []keybinding.Hint {
	return []keybinding.Hint{
		{Keys: "Enter", Description: "new line"},
		{Keys: "Ctrl-Z/Ctrl-Y", Description: "undo/redo"},
		{Keys: "Esc", Description: "cancel"},
	}
}

func (a *textArea) MouseHandler() mouseHandler {
	return keepMouseOwner(a, a.TextArea.MouseHandler())
}

func (a *textArea) Value() string {
	return a.GetText()
}

func (a *textArea) SetValue(value string) {
	a.SetText(value, false)
}

func (a *textArea) SetLabelWidth(width int) {
	a.TextArea.SetLabelWidth(width)
}

func (a *textArea) SetDoneFunc(handler func(key tcell.Key)) {
	a.TextArea.SetFinishedFunc(handler)
}

func (a *textArea) showFocused() {
	a.applyBackground(a.theme.ActiveFieldBackground)
}

func (a *textArea) showIdle() {
	a.applyBackground(a.theme.FieldBackground)
}

func (a *textArea) applyBackground(background tcell.Color) {
	a.SetBackgroundColor(background)
	a.SetTextStyle(tcell.StyleDefault.
		Foreground(a.theme.FieldTextColor).
		Background(background))
}
