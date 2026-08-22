package components

import (
	"ditdah/internal/tui/keybinding"

	"github.com/mattn/go-runewidth"
	"github.com/rivo/tview"
)

const menuBarGap = 2

// MenuBar is one fixed horizontal application menu. Its first element opens
// the supplied menu items; the remaining elements invoke bindings directly.
type MenuBar interface {
	tview.Primitive
	Button() Menu
	Width() int
}

type menuBar struct {
	*tview.Flex
	button Menu
	width  int
}

func newMenuBar(
	controls Factory,
	theme Theme,
	changed func(),
	label string,
	items []MenuItem,
	bindings []keybinding.Binding,
) MenuBar {
	button := controls.Menu(label, items)
	bindings = keybinding.Visible(bindings)
	buttonWidth := runewidth.StringWidth(label) + 2*menuBarGap
	actionsWidth := 0
	bar := &menuBar{
		Flex:   tview.NewFlex(),
		button: button,
		width:  buttonWidth,
	}
	bar.SetBackgroundColor(theme.Background)
	bar.AddItem(button, buttonWidth, 0, false)
	for index, binding := range bindings {
		rightPadding := 0
		if index < len(bindings)-1 {
			rightPadding = menuBarGap
		}
		action := newMenuAction(binding, rightPadding, theme, changed)
		actionWidth := action.Width()
		actionsWidth += actionWidth
		bar.AddItem(action, actionWidth, 0, false)
	}
	bar.width += actionsWidth
	return bar
}

func (m *menuBar) Button() Menu { return m.button }

func (m *menuBar) Width() int { return m.width }
