package components

import (
	"ditdah/internal/tui/keybinding"
	"strings"

	"github.com/gdamore/tcell/v2"
	"github.com/mattn/go-runewidth"
	"github.com/rivo/tview"
)

const (
	menuItemHorizontalPadding = 2
	menuItemShortcutGap       = 2
)

// MenuItem is one action or separator in a Menu.
type MenuItem struct {
	Label       string
	Hotkey      rune
	Description string
	Action      func()
	Separator   bool
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
		style = style.Bold(true)
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
		if (action != tview.MouseLeftClick &&
			action != tview.MouseLeftDoubleClick) || !m.InRect(x, y) {
			return false, nil
		}
		setFocus(m)
		if m.open {
			m.closePopup()
		} else {
			m.openPopup()
		}
		return true, nil
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
	item := m.items[index]
	if item.Separator {
		return
	}
	m.closePopup()
	if item.Action != nil {
		item.Action()
	}
}

type menuPopup struct {
	*tview.Box
	menu       *menu
	bindings   []keybinding.Binding
	selected   int
	itemOffset int
	x          int
	y          int
	width      int
	height     int
}

func newMenuPopup(menu *menu) tview.Primitive {
	popup := &menuPopup{
		Box: tview.NewBox().
			SetBackgroundColor(menu.listBackground).
			SetBorder(true).
			SetBorderColor(menu.borderColor),
		menu:     menu,
		selected: -1,
	}
	popup.bindings = append(popup.bindings, keybinding.OnKey(
		tcell.KeyEscape,
		"close",
		menu.closePopup,
	))
	for index, item := range menu.items {
		if item.Separator || item.Hotkey == 0 {
			continue
		}
		itemIndex := index
		popup.bindings = append(
			popup.bindings,
			keybinding.OnRune(
				item.Hotkey,
				item.Description,
				func() { menu.selectItem(itemIndex) },
			),
		)
	}
	popup.selected = popup.nextItem(-1, 1)
	return popup
}

func (p *menuPopup) KeyBindings() []keybinding.Binding {
	return p.bindings
}

func (p *menuPopup) Draw(screen tcell.Screen) {
	p.position(screen)
	if p.width < 3 || p.height < 3 {
		return
	}
	p.Box.SetRect(p.x, p.y, p.width, p.height)
	p.drawFrame(screen)
	p.updateItemOffset()

	innerWidth := p.width - 2
	visibleRows := p.height - 2
	for row := 0; row < visibleRows; row++ {
		index := p.itemOffset + row
		if index >= len(p.menu.items) {
			break
		}
		y := p.y + 1 + row
		item := p.menu.items[index]
		if item.Separator {
			p.drawSeparator(screen, y)
			continue
		}

		style := tcell.StyleDefault.
			Foreground(p.menu.listText).
			Background(p.menu.listBackground)
		if index == p.selected {
			style = tcell.StyleDefault.
				Foreground(p.menu.selectionText).
				Background(p.menu.selectionBackground).
				Bold(true)
		}
		fill(screen, p.x+1, y, innerWidth, style)
		drawText(screen, menuItemText(item, innerWidth), p.x+1, y, innerWidth, style)
	}
}

func (p *menuPopup) drawFrame(screen tcell.Screen) {
	background := tcell.StyleDefault.Background(p.menu.listBackground)
	for y := p.y; y < p.y+p.height; y++ {
		fill(screen, p.x, y, p.width, background)
	}
	border := tcell.StyleDefault.
		Foreground(p.menu.borderColor).
		Background(p.menu.listBackground)
	for x := p.x + 1; x < p.x+p.width-1; x++ {
		screen.SetContent(x, p.y, tview.Borders.Horizontal, nil, border)
		screen.SetContent(x, p.y+p.height-1, tview.Borders.Horizontal, nil, border)
	}
	for y := p.y + 1; y < p.y+p.height-1; y++ {
		screen.SetContent(p.x, y, tview.Borders.Vertical, nil, border)
		screen.SetContent(p.x+p.width-1, y, tview.Borders.Vertical, nil, border)
	}
	screen.SetContent(p.x, p.y, tview.Borders.TopLeft, nil, border)
	screen.SetContent(p.x+p.width-1, p.y, tview.Borders.TopRight, nil, border)
	screen.SetContent(p.x, p.y+p.height-1, tview.Borders.BottomLeft, nil, border)
	screen.SetContent(p.x+p.width-1, p.y+p.height-1, tview.Borders.BottomRight, nil, border)
}

func menuItemText(item MenuItem, width int) string {
	shortcut := menuItemShortcut(item)
	labelWidth := runewidth.StringWidth(item.Label)
	shortcutWidth := runewidth.StringWidth(shortcut)
	contentWidth := width - 2*menuItemHorizontalPadding
	gap := max(menuItemShortcutGap, contentWidth-labelWidth-shortcutWidth)
	padding := strings.Repeat(" ", menuItemHorizontalPadding)
	return padding + item.Label + strings.Repeat(" ", gap) + shortcut + padding
}

func (p *menuPopup) drawSeparator(screen tcell.Screen, y int) {
	style := tcell.StyleDefault.
		Foreground(p.menu.borderColor).
		Background(p.menu.listBackground)
	screen.SetContent(p.x, y, tview.Borders.LeftT, nil, style)
	for x := p.x + 1; x < p.x+p.width-1; x++ {
		screen.SetContent(x, y, tview.Borders.Horizontal, nil, style)
	}
	screen.SetContent(p.x+p.width-1, y, tview.Borders.RightT, nil, style)
}

func (p *menuPopup) updateItemOffset() {
	visibleRows := p.height - 2
	if p.selected < p.itemOffset {
		p.itemOffset = p.selected
	}
	if p.selected >= p.itemOffset+visibleRows {
		p.itemOffset = p.selected - visibleRows + 1
	}
	p.itemOffset = max(0, min(p.itemOffset, len(p.menu.items)-visibleRows))
}

func (p *menuPopup) position(screen tcell.Screen) {
	screenWidth, screenHeight := screen.Size()
	menuX, menuY, menuWidth, _ := p.menu.GetInnerRect()
	wantedWidth := menuWidth
	for _, item := range p.menu.items {
		if item.Separator {
			continue
		}
		shortcut := menuItemShortcut(item)
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

func menuItemShortcut(item MenuItem) string {
	if item.Hotkey == 0 {
		return ""
	}
	return keybinding.OnRune(item.Hotkey, item.Description, nil).Hint().Keys
}

func (p *menuPopup) Focus(_ func(tview.Primitive)) {
	p.Box.Focus(nil)
}

func (p *menuPopup) Blur() {
	p.Box.Blur()
	p.menu.removePopup()
}

func (p *menuPopup) HasFocus() bool {
	return p.Box.HasFocus()
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
		switch event.Key() {
		case tcell.KeyTab, tcell.KeyDown:
			p.selectNext(1)
		case tcell.KeyBacktab, tcell.KeyUp:
			p.selectNext(-1)
		case tcell.KeyHome, tcell.KeyPgUp:
			p.selected = p.nextItem(-1, 1)
		case tcell.KeyEnd, tcell.KeyPgDn:
			p.selected = p.nextItem(len(p.menu.items), -1)
		case tcell.KeyEnter:
			p.menu.selectItem(p.selected)
		case tcell.KeyRune:
			if event.Rune() == ' ' {
				p.menu.selectItem(p.selected)
			}
		}
	})
}

func (p *menuPopup) selectNext(direction int) {
	if next := p.nextItem(p.selected, direction); next >= 0 {
		p.selected = next
	}
}

func (p *menuPopup) nextItem(from int, direction int) int {
	for index := from + direction; index >= 0 && index < len(p.menu.items); index += direction {
		if !p.menu.items[index].Separator {
			return index
		}
	}
	return -1
}

func (p *menuPopup) MouseHandler() mouseHandler {
	return p.WrapMouseHandler(func(
		action tview.MouseAction,
		event *tcell.EventMouse,
		_ func(tview.Primitive),
	) (bool, tview.Primitive) {
		if !p.menu.open {
			return true, nil
		}
		x, y := event.Position()
		inside := x >= p.x && x < p.x+p.width &&
			y >= p.y && y < p.y+p.height
		if inside {
			if action == tview.MouseLeftClick ||
				action == tview.MouseLeftDoubleClick {
				itemIndex := p.itemOffset + y - p.y - 1
				if itemIndex >= 0 && itemIndex < len(p.menu.items) &&
					!p.menu.items[itemIndex].Separator {
					p.selected = itemIndex
					p.menu.selectItem(itemIndex)
				}
			}
			return true, nil
		}
		if action == tview.MouseLeftClick ||
			action == tview.MouseLeftDoubleClick {
			p.menu.closePopup()
			if p.menu.InRect(x, y) {
				return true, nil
			}
			// Let the completed click reach another control, such as F1/F2.
			return false, nil
		}
		// Only a completed click dismisses the popup.
		return true, nil
	})
}
