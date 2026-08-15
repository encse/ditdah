package components

import (
	"morsemanual/internal/tui/keybinding"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

// InputField is an application-styled text input.
type InputField interface {
	tview.FormItem
	keybinding.HintProvider
	keybinding.ParentBindingBlocker
	Value() string
	SetValue(value string)
	SetPlaceholder(placeholder string)
	SetChangedFunc(handler func(value string))
	SetDoneFunc(handler func(key tcell.Key))
}

func (i *inputField) BlocksParentBindings() bool {
	return true
}

func (i *inputField) KeyHints() []keybinding.Hint {
	return []keybinding.Hint{
		{Keys: "Enter", Description: "done"},
		{Keys: "Esc", Description: "cancel"},
		{Keys: "Tab/Shift+Tab", Description: "next/previous"},
	}
}

func (i *inputField) MouseHandler() mouseHandler {
	return keepMouseOwner(i, i.InputField.MouseHandler())
}

type inputField struct {
	*tview.InputField

	idleBackground  tcell.Color
	focusBackground tcell.Color
}

func newInputField(
	label string,
	value string,
	theme Theme,
	focusChanged func(),
) InputField {
	field := &inputField{
		InputField: tview.NewInputField().
			SetLabel(label).
			SetText(value).
			SetLabelColor(theme.LabelColor).
			SetFieldTextColor(theme.FieldTextColor).
			SetFieldBackgroundColor(theme.FieldBackground).
			SetPlaceholderTextColor(theme.MutedText).
			SetFieldWidth(0),
		idleBackground:  theme.FieldBackground,
		focusBackground: theme.ActiveFieldBackground,
	}
	field.InputField.
		SetFocusFunc(func() {
			field.showFocused()
			notify(focusChanged)
		}).
		SetBlurFunc(field.showIdle)
	return field
}

func (i *inputField) Value() string {
	return i.GetText()
}

func (i *inputField) SetValue(value string) {
	i.SetText(value)
}

func (i *inputField) SetPlaceholder(placeholder string) {
	i.InputField.SetPlaceholder(placeholder)
}

func (i *inputField) SetChangedFunc(handler func(value string)) {
	i.InputField.SetChangedFunc(handler)
}

func (i *inputField) SetDoneFunc(handler func(key tcell.Key)) {
	i.InputField.SetDoneFunc(handler)
}

func (i *inputField) SetFormAttributes(
	labelWidth int,
	labelColor tcell.Color,
	backgroundColor tcell.Color,
	fieldTextColor tcell.Color,
	fieldBackgroundColor tcell.Color,
) tview.FormItem {
	i.idleBackground = fieldBackgroundColor
	i.InputField.SetFormAttributes(
		labelWidth,
		labelColor,
		backgroundColor,
		fieldTextColor,
		fieldBackgroundColor,
	)
	if i.HasFocus() {
		i.showFocused()
	} else {
		i.showIdle()
	}
	return i
}

func (i *inputField) showFocused() {
	i.InputField.SetFieldBackgroundColor(i.focusBackground)
}

func (i *inputField) showIdle() {
	i.InputField.SetFieldBackgroundColor(i.idleBackground)
}
