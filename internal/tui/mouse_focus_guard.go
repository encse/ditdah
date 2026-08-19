package tui

import (
	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

// mouseFocusGuard enforces the application's declared focus model for mouse
// interactions. Controls may consume clicks, but only declared focusables and
// active overlays may receive focus or retain mouse capture.
type mouseFocusGuard struct {
	tview.Primitive
	allowed func(tview.Primitive) bool
}

func newMouseFocusGuard(
	primitive tview.Primitive,
	allowed func(tview.Primitive) bool,
) tview.Primitive {
	return &mouseFocusGuard{Primitive: primitive, allowed: allowed}
}

func (g *mouseFocusGuard) MouseHandler() func(
	tview.MouseAction,
	*tcell.EventMouse,
	func(tview.Primitive),
) (bool, tview.Primitive) {
	handler := g.Primitive.MouseHandler()
	if handler == nil {
		return nil
	}
	return func(
		action tview.MouseAction,
		event *tcell.EventMouse,
		setFocus func(tview.Primitive),
	) (bool, tview.Primitive) {
		consumed, capture := handler(
			action,
			event,
			func(primitive tview.Primitive) {
				if g.isAllowed(primitive) {
					setFocus(primitive)
				}
			},
		)
		if capture == nil || !g.isAllowed(capture) {
			return consumed, nil
		}
		return consumed, newMouseFocusGuard(capture, g.allowed)
	}
}

func (g *mouseFocusGuard) isAllowed(primitive tview.Primitive) bool {
	return primitive != nil && g.allowed != nil && g.allowed(primitive)
}
