// Package components provides consistently styled tview form controls.
package components

import (
	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

// Overlay is a displayed overlay which can remove itself from its host.
type Overlay interface {
	Close()
}

// OverlayHost displays primitives above the current user interface. Hosts may
// stack overlays, so a select popup can be opened from inside a modal dialog.
type OverlayHost interface {
	Push(primitive tview.Primitive) Overlay
}

// Theme contains only the colors needed by the component package.
type Theme struct {
	Background             tcell.Color
	PrimaryText            tcell.Color
	SecondaryText          tcell.Color
	MutedText              tcell.Color
	Accent                 tcell.Color
	Border                 tcell.Color
	LabelColor             tcell.Color
	FieldTextColor         tcell.Color
	FieldBackground        tcell.Color
	ActiveFieldBackground  tcell.Color
	CursorColor            tcell.Color
	SelectionText          tcell.Color
	SelectionBackground    tcell.Color
	PopupBorder            tcell.Color
	ButtonText             tcell.Color
	ButtonBackground       tcell.Color
	ActiveButtonText       tcell.Color
	ActiveButtonBackground tcell.Color
}

// Factory creates controls which share one theme.
type Factory interface {
	Modal() Factory
	Header() Header
	Footer() Footer
	Button(label string) Button
	InputField(label, value string) InputField
	TextArea(label, value string) TextArea
	TextView() TextView
	Table(title string) Table
	SelectField(
		label string,
		options []string,
		selected int,
		labelWidth int,
		fieldWidth int,
	) SelectField
}

type factory struct {
	theme        Theme
	modalTheme   Theme
	overlays     OverlayHost
	focusChanged func()
}

// Dependencies are shared by every control created by a Factory.
type Dependencies struct {
	Theme        Theme
	ModalTheme   Theme
	Overlays     OverlayHost
	FocusChanged func()
}

// New creates a component factory with shared dependencies.
func New(dependencies Dependencies) Factory {
	modalTheme := dependencies.ModalTheme
	if modalTheme == (Theme{}) {
		modalTheme = dependencies.Theme
	}
	return factory{
		theme:        dependencies.Theme,
		modalTheme:   modalTheme,
		overlays:     dependencies.Overlays,
		focusChanged: dependencies.FocusChanged,
	}
}

func (f factory) Modal() Factory {
	f.theme = f.modalTheme
	return f
}

func (f factory) Header() Header {
	return newHeader(f)
}

func (f factory) Footer() Footer {
	return newFooter(f)
}

func (f factory) Button(label string) Button {
	return newButton(label, f.theme, f.focusChanged)
}

func (f factory) InputField(label, value string) InputField {
	return newInputField(label, value, f.theme, f.focusChanged)
}

func (f factory) TextArea(label, value string) TextArea {
	return newTextArea(label, value, f.theme, f.focusChanged)
}

func (f factory) TextView() TextView {
	return newTextView(f.theme, f.focusChanged)
}

func (f factory) Table(title string) Table {
	return newTable(title, f.theme, f.focusChanged)
}

func (f factory) SelectField(
	label string,
	options []string,
	selected int,
	labelWidth int,
	fieldWidth int,
) SelectField {
	return newSelectField(
		label,
		options,
		selected,
		labelWidth,
		fieldWidth,
		f.theme,
		f.overlays,
		f.focusChanged,
	)
}

func notify(handler func()) {
	if handler != nil {
		handler()
	}
}

type mouseHandler = func(
	action tview.MouseAction,
	event *tcell.EventMouse,
	setFocus func(tview.Primitive),
) (consumed bool, capture tview.Primitive)

// keepMouseOwner delegates mouse behaviour to an embedded tview primitive
// without allowing that primitive to replace its application wrapper as the
// focused or captured control.
func keepMouseOwner(owner tview.Primitive, handler mouseHandler) mouseHandler {
	if handler == nil {
		return nil
	}
	return func(
		action tview.MouseAction,
		event *tcell.EventMouse,
		setFocus func(tview.Primitive),
	) (bool, tview.Primitive) {
		consumed, capture := handler(action, event, func(tview.Primitive) {
			setFocus(owner)
		})
		if capture != nil {
			capture = owner
		}
		return consumed, capture
	}
}
