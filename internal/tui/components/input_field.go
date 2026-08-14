package components

import (
	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

// InputField is an application-styled text input.
type InputField interface {
	tview.FormItem
	Value() string
	SetValue(value string)
}

type inputField struct {
	*tview.InputField

	idleBackground  tcell.Color
	focusBackground tcell.Color
}

func newInputField(label, value string, theme Theme) InputField {
	field := &inputField{
		InputField: tview.NewInputField().
			SetLabel(label).
			SetText(value).
			SetLabelColor(theme.LabelColor).
			SetFieldTextColor(theme.FieldTextColor).
			SetFieldBackgroundColor(theme.FieldBackground).
			SetFieldWidth(0),
		idleBackground:  theme.FieldBackground,
		focusBackground: theme.ActiveFieldBackground,
	}
	field.InputField.
		SetFocusFunc(field.showFocused).
		SetBlurFunc(field.showIdle)
	return field
}

func (i *inputField) Value() string {
	return i.GetText()
}

func (i *inputField) SetValue(value string) {
	i.SetText(value)
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
