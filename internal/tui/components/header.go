package components

import (
	"github.com/rivo/tview"
)

// Header is the structured top-level application header. The title and status
// are owned by the header; the menu is an independent component placed on the
// left side.
type Header interface {
	tview.Primitive
	SetTitle(title string)
	SetMenu(menu tview.Primitive, width int)
	SetStatus(status string)
}

type header struct {
	*tview.Flex
	title     TextView
	emptyMenu TextView
	menu      tview.Primitive
	menuWidth int
	status    TextView
}

func newHeader(controls Factory) Header {
	title := controls.TextView()
	title.SetStyle(TextViewAccent)
	title.SetTextAlign(tview.AlignCenter)

	status := controls.TextView()
	status.SetTextAlign(tview.AlignRight)
	status.SetStyle(TextViewMuted)

	header := &header{
		Flex:      tview.NewFlex(),
		title:     title,
		emptyMenu: controls.TextView(),
		status:    status,
	}
	header.rebuild()
	return header
}

func (h *header) SetTitle(title string) {
	h.title.SetText(title)
}

func (h *header) SetMenu(menu tview.Primitive, width int) {
	h.menu = menu
	h.menuWidth = max(0, width)
	h.rebuild()
}

func (h *header) SetStatus(status string) {
	h.status.SetText(status)
}

func (h *header) rebuild() {
	menu := h.menu
	if menu == nil {
		menu = h.emptyMenu
	}
	h.Clear().
		AddItem(menu, h.menuWidth, 0, false).
		AddItem(h.title, 0, 1, false).
		AddItem(h.status, h.menuWidth, 0, false)
}
