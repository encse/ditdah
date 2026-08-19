package tui

import (
	"testing"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

func TestMouseFocusGuardRejectsUndeclaredFocusAndCapture(t *testing.T) {
	forbidden := tview.NewBox()
	source := &mouseRequestPrimitive{
		Box:     tview.NewBox(),
		focus:   forbidden,
		capture: forbidden,
	}
	guard := newMouseFocusGuard(source, func(tview.Primitive) bool {
		return false
	})
	var focused tview.Primitive
	consumed, capture := guard.MouseHandler()(
		tview.MouseLeftDown,
		tcell.NewEventMouse(0, 0, tcell.Button1, 0),
		func(primitive tview.Primitive) { focused = primitive },
	)
	if !consumed {
		t.Fatal("guarded event was not consumed")
	}
	if focused != nil {
		t.Fatalf("focused primitive = %T, want nil", focused)
	}
	if capture != nil {
		t.Fatalf("capture = %T, want nil", capture)
	}
}

func TestMouseFocusGuardKeepsAllowedCaptureGuarded(t *testing.T) {
	source := &mouseRequestPrimitive{Box: tview.NewBox()}
	source.focus = source
	source.capture = source
	guard := newMouseFocusGuard(source, func(primitive tview.Primitive) bool {
		return primitive == source
	})
	var focused tview.Primitive
	_, capture := guard.MouseHandler()(
		tview.MouseLeftDown,
		tcell.NewEventMouse(0, 0, tcell.Button1, 0),
		func(primitive tview.Primitive) { focused = primitive },
	)
	if focused != source {
		t.Fatalf("focused primitive = %T, want source", focused)
	}
	if _, ok := capture.(*mouseFocusGuard); !ok {
		t.Fatalf("capture = %T, want guarded capture", capture)
	}
}

type mouseRequestPrimitive struct {
	*tview.Box
	focus   tview.Primitive
	capture tview.Primitive
}

func (p *mouseRequestPrimitive) MouseHandler() func(
	tview.MouseAction,
	*tcell.EventMouse,
	func(tview.Primitive),
) (bool, tview.Primitive) {
	return func(
		_ tview.MouseAction,
		_ *tcell.EventMouse,
		setFocus func(tview.Primitive),
	) (bool, tview.Primitive) {
		setFocus(p.focus)
		return true, p.capture
	}
}
