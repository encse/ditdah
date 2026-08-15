package components

import (
	"io"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

// TextViewStyle selects one of the application text styles.
type TextViewStyle int

const (
	TextViewPrimary TextViewStyle = iota
	TextViewSecondary
	TextViewMuted
	TextViewAccent
)

// TextView is an application-styled text display and writer.
type TextView interface {
	tview.Primitive
	io.Writer
	Clear()
	Text() string
	SetText(text string)
	SetStyle(style TextViewStyle)
	SetTextColor(color tcell.Color)
	SetTextAlign(alignment int)
	SetDynamicColors(enabled bool)
	SetWordWrap(enabled bool)
	SetBorder(title string)
}

type textView struct {
	*tview.TextView
	theme Theme
}

func newTextView(theme Theme) TextView {
	view := tview.NewTextView().
		SetTextColor(theme.PrimaryText)
	view.SetBackgroundColor(theme.Background)
	return &textView{TextView: view, theme: theme}
}

func (v *textView) Clear() {
	v.TextView.Clear()
}

func (v *textView) Text() string {
	return v.GetText(false)
}

func (v *textView) SetText(text string) {
	v.TextView.SetText(text)
}

func (v *textView) SetStyle(style TextViewStyle) {
	var color tcell.Color
	switch style {
	case TextViewSecondary:
		color = v.theme.SecondaryText
	case TextViewMuted:
		color = v.theme.MutedText
	case TextViewAccent:
		color = v.theme.Accent
	default:
		color = v.theme.PrimaryText
	}
	v.TextView.SetTextColor(color)
}

func (v *textView) SetTextColor(color tcell.Color) {
	v.TextView.SetTextColor(color)
}

func (v *textView) SetTextAlign(alignment int) {
	v.TextView.SetTextAlign(alignment)
}

func (v *textView) SetDynamicColors(enabled bool) {
	v.TextView.SetDynamicColors(enabled)
}

func (v *textView) SetWordWrap(enabled bool) {
	v.TextView.SetWordWrap(enabled)
}

func (v *textView) SetBorder(title string) {
	v.TextView.SetBorder(true).
		SetBorderColor(v.theme.Border).
		SetTitle(title).
		SetTitleColor(v.theme.Accent)
}
