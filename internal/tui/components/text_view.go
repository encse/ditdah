package components

import (
	"io"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

// TextView is an application-styled text display and writer.
type TextView interface {
	tview.Primitive
	io.Writer
	Clear()
	SetText(text string)
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

func (v *textView) SetText(text string) {
	v.TextView.SetText(text)
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
