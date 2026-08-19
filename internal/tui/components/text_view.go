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
	TextViewDanger
)

// TextView is an application-styled text display and writer.
type TextView interface {
	tview.Primitive
	io.Writer
	Clear()
	InnerRect() (x int, y int, width int, height int)
	ScrollOffset() (row int, column int)
	ScrollToStart()
	ScrollToEnd()
	Text() string
	SetText(text string)
	SetStyle(style TextViewStyle)
	SetTextColor(color tcell.Color)
	SetTextAlign(alignment int)
	SetDynamicColors(enabled bool)
	SetScrollable(enabled bool)
	SetWrap(enabled bool)
	SetWordWrap(enabled bool)
	SetBorder(title string)
}

type textView struct {
	*tview.TextView
	theme Theme
}

func (v *textView) MouseHandler() mouseHandler {
	return keepMouseOwner(v, v.TextView.MouseHandler())
}

func newTextView(theme Theme, focusChanged func()) TextView {
	view := tview.NewTextView().
		SetTextColor(theme.PrimaryText)
	view.SetBackgroundColor(theme.Background)
	view.SetFocusFunc(func() {
		notify(focusChanged)
	})
	return &textView{TextView: view, theme: theme}
}

func (v *textView) Clear() {
	v.TextView.Clear()
}

func (v *textView) InnerRect() (int, int, int, int) {
	return v.GetInnerRect()
}

func (v *textView) ScrollOffset() (int, int) {
	return v.GetScrollOffset()
}

func (v *textView) ScrollToStart() {
	v.ScrollToBeginning()
}

func (v *textView) ScrollToEnd() {
	v.TextView.ScrollToEnd()
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
	case TextViewDanger:
		color = v.theme.DangerText
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

func (v *textView) SetScrollable(enabled bool) {
	v.TextView.SetScrollable(enabled)
}

func (v *textView) SetWrap(enabled bool) {
	v.TextView.SetWrap(enabled)
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
