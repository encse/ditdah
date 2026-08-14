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
	LabelColor            tcell.Color
	FieldTextColor        tcell.Color
	FieldBackground       tcell.Color
	ActiveFieldBackground tcell.Color
	SelectionText         tcell.Color
	SelectionBackground   tcell.Color
	PopupBorder           tcell.Color
}

// Factory creates controls which share one theme.
type Factory interface {
	InputField(label, value string) InputField
	SelectField(
		label string,
		options []string,
		selected int,
		labelWidth int,
		fieldWidth int,
	) SelectField
}

type factory struct {
	theme    Theme
	overlays OverlayHost
}

// Dependencies are shared by every control created by a Factory.
type Dependencies struct {
	Theme    Theme
	Overlays OverlayHost
}

// New creates a component factory with shared dependencies.
func New(dependencies Dependencies) Factory {
	return factory{
		theme:    dependencies.Theme,
		overlays: dependencies.Overlays,
	}
}

func (f factory) InputField(label, value string) InputField {
	return newInputField(label, value, f.theme)
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
	)
}
