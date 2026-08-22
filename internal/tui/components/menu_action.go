package components

import (
	"ditdah/internal/tui/keybinding"

	"github.com/gdamore/tcell/v2"
	"github.com/mattn/go-runewidth"
	"github.com/rivo/tview"
)

type menuAction struct {
	*tview.TextView
	binding keybinding.Binding
	width   int
	changed func()
}

func newMenuAction(
	binding keybinding.Binding,
	rightPadding int,
	theme Theme,
	changed func(),
) *menuAction {
	hint := binding.Hint()
	label := hint.Keys + " " + hint.Description
	view := tview.NewTextView().
		SetText(label).
		SetTextColor(theme.PrimaryText).
		SetTextAlign(tview.AlignLeft).
		SetWrap(false).
		SetWordWrap(false)
	view.SetBackgroundColor(theme.Background)
	return &menuAction{
		TextView: view,
		binding:  binding,
		width:    runewidth.StringWidth(label) + max(0, rightPadding),
		changed:  changed,
	}
}

func (a *menuAction) Width() int { return a.width }

func (a *menuAction) MouseHandler() mouseHandler {
	return a.WrapMouseHandler(func(
		action tview.MouseAction,
		event *tcell.EventMouse,
		_ func(tview.Primitive),
	) (bool, tview.Primitive) {
		if action != tview.MouseLeftClick &&
			action != tview.MouseLeftDoubleClick {
			return false, nil
		}
		x, y := event.Position()
		if !a.InRect(x, y) {
			return false, nil
		}
		a.binding.Invoke()
		notify(a.changed)
		return true, nil
	})
}
