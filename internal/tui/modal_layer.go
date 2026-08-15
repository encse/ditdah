package tui

import (
	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

type modalLayer struct {
	*tview.Box
	content            tview.Primitive
	preferredWidth     int
	preferredHeight    int
	backdropForeground tcell.Color
	backdropBackground tcell.Color
}

func newModalLayer(
	content tview.Primitive,
	width int,
	height int,
	foreground tcell.Color,
	background tcell.Color,
) *modalLayer {
	return &modalLayer{
		Box:                tview.NewBox(),
		content:            content,
		preferredWidth:     width,
		preferredHeight:    height,
		backdropForeground: foreground,
		backdropBackground: background,
	}
}

func (l *modalLayer) Draw(screen tcell.Screen) {
	x, y, width, height := l.GetRect()
	for row := y; row < y+height; row++ {
		for column := x; column < x+width; column++ {
			main, combining, style, _ := screen.GetContent(column, row)
			style = style.
				Foreground(l.backdropForeground).
				Background(l.backdropBackground)
			screen.SetContent(column, row, main, combining, style)
		}
	}

	dialogWidth := modalDimension(l.preferredWidth, width)
	dialogHeight := modalDimension(l.preferredHeight, height)
	dialogX := x + (width-dialogWidth)/2
	dialogY := y + (height-dialogHeight)/2
	l.content.SetRect(dialogX, dialogY, dialogWidth, dialogHeight)
	l.content.Draw(screen)
}

func (l *modalLayer) Focus(delegate func(tview.Primitive)) {
	if delegate == nil {
		l.content.Focus(nil)
		return
	}
	delegate(l.content)
}

func (l *modalLayer) HasFocus() bool {
	return l.content.HasFocus() || l.Box.HasFocus()
}

func (l *modalLayer) Blur() {
	l.content.Blur()
	l.Box.Blur()
}

func (l *modalLayer) InputHandler() func(
	*tcell.EventKey,
	func(tview.Primitive),
) {
	return l.WrapInputHandler(func(
		event *tcell.EventKey,
		setFocus func(tview.Primitive),
	) {
		if handler := l.content.InputHandler(); handler != nil {
			handler(event, setFocus)
		}
	})
}

func (l *modalLayer) PasteHandler() func(string, func(tview.Primitive)) {
	return l.WrapPasteHandler(func(
		text string,
		setFocus func(tview.Primitive),
	) {
		if handler := l.content.PasteHandler(); handler != nil {
			handler(text, setFocus)
		}
	})
}

func (l *modalLayer) MouseHandler() func(
	tview.MouseAction,
	*tcell.EventMouse,
	func(tview.Primitive),
) (bool, tview.Primitive) {
	return l.WrapMouseHandler(func(
		action tview.MouseAction,
		event *tcell.EventMouse,
		setFocus func(tview.Primitive),
	) (bool, tview.Primitive) {
		x, y := event.Position()
		contentX, contentY, width, height := l.content.GetRect()
		inside := x >= contentX && x < contentX+width &&
			y >= contentY && y < contentY+height
		if inside {
			if handler := l.content.MouseHandler(); handler != nil {
				consumed, capture := handler(action, event, setFocus)
				if consumed {
					return true, capture
				}
			}
		}
		return true, nil
	})
}

func modalDimension(preferred int, available int) int {
	if available <= 0 {
		return 0
	}
	if preferred <= 0 || preferred > available-4 {
		preferred = available - 4
	}
	if preferred < 1 {
		return available
	}
	return preferred
}
