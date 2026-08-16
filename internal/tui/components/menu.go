package components

import (
	"morsemanual/internal/tui/keybinding"
	"strings"

	"github.com/gdamore/tcell/v2"
	"github.com/mattn/go-runewidth"
	"github.com/rivo/tview"
)

const (
	menuItemHorizontalPadding = 2
	menuItemShortcutGap       = 2
)

// MenuItem is one selectable action in a Menu.
type MenuItem struct {
	Label   string
	Binding keybinding.Binding
}

// Menu is a menu-bar item which opens its actions in an overlay list.
type Menu interface {
	tview.Primitive
	keybinding.BindingProvider
}

type menu struct {
	*tview.Box
	label               string
	items               []MenuItem
	textColor           tcell.Color
	background          tcell.Color
	activeText          tcell.Color
	activeBackground    tcell.Color
	listText            tcell.Color
	listBackground      tcell.Color
	selectionText       tcell.Color
	selectionBackground tcell.Color
	borderColor         tcell.Color
	overlays            OverlayHost
	overlay             Overlay
	open                bool
}

func newMenu(
	label string,
	items []MenuItem,
	theme Theme,
	overlays OverlayHost,
	focusChanged func(),
) Menu {
	menu := &menu{
		Box:                 tview.NewBox().SetBackgroundColor(theme.Background),
		label:               label,
		items:               append([]MenuItem(nil), items...),
		textColor:           theme.PrimaryText,
		background:          theme.Background,
		activeText:          theme.SelectionText,
		activeBackground:    theme.SelectionBackground,
		listText:            theme.PrimaryText,
		listBackground:      theme.Background,
		selectionText:       theme.SelectionText,
		selectionBackground: theme.SelectionBackground,
		borderColor:         theme.PopupBorder,
		overlays:            overlays,
	}
	menu.SetFocusFunc(func() { notify(focusChanged) })
	return menu
}

func (m *menu) KeyBindings() []keybinding.Binding {
	return []keybinding.Binding{keybinding.On(
		"open",
		m.openPopup,
		keybinding.Key(tcell.KeyEnter),
		keybinding.Rune(' '),
	)}
}

func (m *menu) Draw(screen tcell.Screen) {
	m.Box.DrawForSubclass(screen, m)
	x, y, width, height := m.GetInnerRect()
	if width < 1 || height < 1 {
		return
	}

	style := tcell.StyleDefault.
		Foreground(m.textColor).
		Background(m.background)
	if m.HasFocus() || m.open {
		style = tcell.StyleDefault.
			Foreground(m.activeText).
			Background(m.activeBackground).
			Bold(true)
	}
	fill(screen, x, y, width, style)

	labelWidth := runewidth.StringWidth(m.label)
	start := x + max(0, (width-labelWidth)/2)
	drawText(screen, m.label, start, y, width, style)
}

func (m *menu) InputHandler() func(
	*tcell.EventKey,
	func(tview.Primitive),
) {
	return m.WrapInputHandler(func(
		event *tcell.EventKey,
		_ func(tview.Primitive),
	) {
		for _, binding := range m.KeyBindings() {
			if binding.Handle(event) {
				return
			}
		}
	})
}

func (m *menu) MouseHandler() mouseHandler {
	return m.WrapMouseHandler(func(
		action tview.MouseAction,
		event *tcell.EventMouse,
		setFocus func(tview.Primitive),
	) (bool, tview.Primitive) {
		x, y := event.Position()
		if action != tview.MouseLeftDown || !m.InRect(x, y) {
			return false, nil
		}
		setFocus(m)
		if m.open {
			m.closePopup()
		} else {
			m.openPopup()
		}
		return true, m
	})
}

func (m *menu) openPopup() {
	if m.open || len(m.items) == 0 || m.overlays == nil {
		return
	}
	m.open = true
	m.overlay = m.overlays.Push(newMenuPopup(m))
}

func (m *menu) closePopup() {
	m.open = false
	if m.overlay == nil {
		return
	}
	overlay := m.overlay
	m.overlay = nil
	overlay.Close()
}

func (m *menu) removePopup() {
	m.open = false
	if m.overlay == nil {
		return
	}
	overlay := m.overlay
	m.overlay = nil
	overlay.Remove()
}

func (m *menu) selectItem(index int) {
	if index < 0 || index >= len(m.items) {
		return
	}
	binding := m.items[index].Binding
	m.closePopup()
	binding.Invoke()
}

type menuPopup struct {
	*tview.Box
	menu   *menu
	list   *tview.List
	x      int
	y      int
	width  int
	height int
}

func newMenuPopup(menu *menu) tview.Primitive {
	list := tview.NewList().
		ShowSecondaryText(false).
		SetHighlightFullLine(true).
		SetWrapAround(false).
		SetMainTextStyle(tcell.StyleDefault.
			Foreground(menu.listText).
			Background(menu.listBackground)).
		SetSelectedStyle(tcell.StyleDefault.
			Foreground(menu.selectionText).
			Background(menu.selectionBackground).
			Bold(true))
	list.SetBackgroundColor(menu.listBackground)
	list.SetBorder(true).SetBorderColor(menu.borderColor)
	for range menu.items {
		list.AddItem("", "", 0, nil)
	}

	popup := &menuPopup{
		Box:  tview.NewBox(),
		menu: menu,
		list: list,
	}
	list.SetSelectedFunc(func(index int, _ string, _ string, _ rune) {
		menu.selectItem(index)
	})
	list.SetDoneFunc(menu.closePopup)
	return popup
}

func (p *menuPopup) KeyBindings() []keybinding.Binding {
	return []keybinding.Binding{keybinding.OnKey(
		tcell.KeyEscape,
		"close",
		p.menu.closePopup,
	)}
}

func (p *menuPopup) Draw(screen tcell.Screen) {
	p.position(screen)
	if p.width < 3 || p.height < 3 {
		return
	}
	p.list.SetRect(p.x, p.y, p.width, p.height)
	p.updateItemText()
	p.list.Draw(screen)
}

func (p *menuPopup) updateItemText() {
	innerWidth := max(0, p.width-2)
	for index, item := range p.menu.items {
		shortcut := item.Binding.Hint().Keys
		labelWidth := runewidth.StringWidth(item.Label)
		shortcutWidth := runewidth.StringWidth(shortcut)
		contentWidth := innerWidth - 2*menuItemHorizontalPadding
		gap := max(menuItemShortcutGap, contentWidth-labelWidth-shortcutWidth)
		padding := strings.Repeat(" ", menuItemHorizontalPadding)
		text := padding + item.Label + strings.Repeat(" ", gap) +
			shortcut + padding
		p.list.SetItemText(index, text, "")
	}
}

func (p *menuPopup) position(screen tcell.Screen) {
	screenWidth, screenHeight := screen.Size()
	menuX, menuY, menuWidth, _ := p.menu.GetInnerRect()
	wantedWidth := menuWidth
	for _, item := range p.menu.items {
		shortcut := item.Binding.Hint().Keys
		wantedWidth = max(
			wantedWidth,
			runewidth.StringWidth(item.Label)+
				runewidth.StringWidth(shortcut)+
				2*menuItemHorizontalPadding+menuItemShortcutGap+2,
		)
	}
	p.width = min(wantedWidth, screenWidth)
	p.x = min(menuX, screenWidth-p.width)

	wantedHeight := len(p.menu.items) + 2
	spaceBelow := screenHeight - menuY - 1
	p.height = min(wantedHeight, spaceBelow)
	p.y = menuY + 1
	if p.height < 3 && menuY >= 3 {
		p.height = min(wantedHeight, menuY)
		p.y = menuY - p.height
	}
}

func (p *menuPopup) Focus(_ func(tview.Primitive)) {
	p.Box.Focus(nil)
}

func (p *menuPopup) Blur() {
	p.Box.Blur()
	p.list.Blur()
	p.menu.removePopup()
}

func (p *menuPopup) HasFocus() bool {
	return p.Box.HasFocus() || p.list.HasFocus()
}

func (p *menuPopup) InputHandler() func(
	*tcell.EventKey,
	func(tview.Primitive),
) {
	return p.WrapInputHandler(func(
		event *tcell.EventKey,
		setFocus func(tview.Primitive),
	) {
		for _, binding := range p.KeyBindings() {
			if binding.Handle(event) {
				return
			}
		}
		if handler := p.list.InputHandler(); handler != nil {
			handler(event, setFocus)
		}
	})
}

func (p *menuPopup) MouseHandler() mouseHandler {
	return p.WrapMouseHandler(func(
		action tview.MouseAction,
		event *tcell.EventMouse,
		_ func(tview.Primitive),
	) (bool, tview.Primitive) {
		x, y := event.Position()
		inside := x >= p.x && x < p.x+p.width &&
			y >= p.y && y < p.y+p.height
		var capture tview.Primitive
		if action == tview.MouseLeftDown {
			capture = p
		}
		if inside {
			if handler := p.list.MouseHandler(); handler != nil {
				consumed, _ := handler(
					action,
					event,
					func(tview.Primitive) {},
				)
				if consumed {
					return true, capture
				}
			}
			return true, capture
		}
		if action == tview.MouseLeftDown || action == tview.MouseLeftClick {
			p.menu.closePopup()
		}
		return true, capture
	})
}
