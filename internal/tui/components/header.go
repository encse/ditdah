package components

import (
	"github.com/rivo/tview"
)

// Header is the structured top-level application header. The title and status
// are owned by the header; the menu is an independent component placed in its
// centre slot.
type Header interface {
	tview.Primitive
	SetTitle(title string)
	SetMenu(menu tview.Primitive)
	SetStatus(status string)
}

type header struct {
	*tview.Flex
	title     TextView
	emptyMenu TextView
	menu      tview.Primitive
	status    TextView
}

func newHeader(controls Factory) Header {
	title := controls.TextView()
	title.SetStyle(TextViewAccent)

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

func (h *header) SetMenu(menu tview.Primitive) {
	h.menu = menu
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
		AddItem(h.title, 0, 1, false).
		AddItem(menu, 0, 2, false).
		AddItem(h.status, 0, 1, false)
}
