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
	SetBindings(bindings ...keybinding.Binding)
}

type textArea struct {
	*tview.TextArea
	theme    Theme
	bindings []keybinding.Binding
}

func (a *textArea) Draw(screen tcell.Screen) {
	a.TextArea.Draw(screen)
	if a.HasFocus() {
		screen.SetCursorStyle(tcell.CursorStyleDefault, a.theme.CursorColor)
	}
}

func (a *textArea) InputHandler() func(
	*tcell.EventKey,
	func(tview.Primitive),
) {
	native := a.TextArea.InputHandler()
	return func(event *tcell.EventKey, setFocus func(tview.Primitive)) {
		for _, binding := range a.bindings {
			if binding.Handle(event) {
				return
			}
		}
		native(event, setFocus)
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
	native := []keybinding.Hint{
		{Keys: "Enter", Description: "new line"},
		{Keys: "Ctrl-Z/Ctrl-Y", Description: "undo/redo"},
	}
	return keybinding.MergeHints(native, keybinding.Hints(a.bindings)...)
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

func (a *textArea) SetBindings(bindings ...keybinding.Binding) {
	a.bindings = append(a.bindings[:0], bindings...)
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
