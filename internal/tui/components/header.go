package components

import "github.com/rivo/tview"

// Header is the structured top-level application header.
type Header interface {
	tview.Primitive
	SetMenu(menu tview.Primitive, width int)
	SetStatus(status string)
}

type header struct {
	*tview.Flex
	spacer    TextView
	emptyMenu TextView
	menu      tview.Primitive
	menuWidth int
	status    TextView
}

func newHeader(controls Factory) Header {
	status := controls.TextView()
	status.SetTextAlign(tview.AlignRight)
	status.SetStyle(TextViewMuted)

	header := &header{
		Flex:      tview.NewFlex(),
		spacer:    controls.TextView(),
		emptyMenu: controls.TextView(),
		status:    status,
	}
	header.rebuild()
	return header
}

func (h *header) SetMenu(menu tview.Primitive, width int) {
	h.menu = menu
	h.menuWidth = max(0, width)
	h.rebuild()
}

func (h *header) SetStatus(status string) {
	oldWidth := tview.TaggedStringWidth(h.status.Text())
	h.status.SetText(status)
	if oldWidth != tview.TaggedStringWidth(status) {
		h.rebuild()
	}
}

func (h *header) rebuild() {
	menu := h.menu
	if menu == nil {
		menu = h.emptyMenu
	}
	h.Clear().AddItem(menu, h.menuWidth, 0, false)
	h.AddItem(h.spacer, 0, 1, false).
		AddItem(h.status, tview.TaggedStringWidth(h.status.Text()), 0, false)
}
