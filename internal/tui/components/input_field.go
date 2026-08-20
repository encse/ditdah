package components

import (
	"morsemanual/internal/tui/keybinding"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

// InputField is an application-styled text input.
type InputField interface {
	tview.FormItem
	keybinding.BindingProvider
	keybinding.ParentBindingBlocker
	Value() string
	SetValue(value string)
	SetLabelWidth(width int)
	SetMaskCharacter(mask rune)
	SetPlaceholder(placeholder string)
	SetChangedFunc(handler func(value string))
	SetBlurFunc(handler func())
	SetBindings(bindings ...keybinding.Binding)
}

func (i *inputField) BlocksParentBindings() bool {
	return true
}

func (i *inputField) KeyBindings() []keybinding.Binding {
	return append([]keybinding.Binding(nil), i.bindings...)
}

func (i *inputField) MouseHandler() mouseHandler {
	return keepMouseOwner(i, i.InputField.MouseHandler())
}

type inputField struct {
	*tview.InputField

	idleBackground  tcell.Color
	focusBackground tcell.Color
	theme           Theme
	bindings        []keybinding.Binding
}

func (i *inputField) Draw(screen tcell.Screen) {
	i.InputField.Draw(screen)
	if i.HasFocus() {
		screen.SetCursorStyle(tcell.CursorStyleDefault, i.theme.CursorColor)
	}
}

func (i *inputField) InputHandler() func(
	*tcell.EventKey,
	func(tview.Primitive),
) {
	native := i.InputField.InputHandler()
	return func(event *tcell.EventKey, setFocus func(tview.Primitive)) {
		for _, binding := range i.bindings {
			if binding.Handle(event) {
				return
			}
		}
		native(event, setFocus)
	}
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
		theme:           theme,
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

func (i *inputField) SetLabelWidth(width int) {
	i.SetFormAttributes(
		width,
		i.theme.LabelColor,
		i.theme.Background,
		i.theme.FieldTextColor,
		i.idleBackground,
	)
}

func (i *inputField) SetMaskCharacter(mask rune) {
	i.InputField.SetMaskCharacter(mask)
}

func (i *inputField) SetPlaceholder(placeholder string) {
	i.InputField.SetPlaceholder(placeholder)
}

func (i *inputField) SetChangedFunc(handler func(value string)) {
	i.InputField.SetChangedFunc(handler)
}

func (i *inputField) SetBlurFunc(handler func()) {
	i.InputField.SetBlurFunc(func() {
		i.showIdle()
		if handler != nil {
			handler()
		}
	})
}

func (i *inputField) SetBindings(bindings ...keybinding.Binding) {
	i.bindings = append(i.bindings[:0], bindings...)
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
