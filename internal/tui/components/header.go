package components

import (
	"strings"

	"github.com/mattn/go-runewidth"
	"github.com/rivo/tview"
)

const maxHeaderStatusWidth = 64

// Header is the structured top-level application header.
type Header interface {
	tview.Primitive
	SetMenu(menu tview.Primitive, width int)
	SetStatus(status string)
}

type header struct {
	*tview.Flex
	spacer     TextView
	emptyMenu  TextView
	menu       tview.Primitive
	menuWidth  int
	status     TextView
	statusText string
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
	h.statusText = status
	h.rebuild()
}

func (h *header) SetRect(x, y, width, height int) {
	h.Flex.SetRect(x, y, width, height)
	h.rebuild()
}

func (h *header) rebuild() {
	menu := h.menu
	if menu == nil {
		menu = h.emptyMenu
	}
	_, _, width, _ := h.GetRect()
	statusWidth := maxHeaderStatusWidth
	if width > 0 {
		statusWidth = min(statusWidth, max(0, width-h.menuWidth-1))
	}
	status := truncateHeaderStatus(h.statusText, statusWidth)
	h.status.SetText(status)

	h.Clear().AddItem(menu, h.menuWidth, 0, false)
	h.AddItem(h.spacer, 0, 1, false).
		AddItem(h.status, runewidth.StringWidth(status), 0, false)
}

func truncateHeaderStatus(status string, width int) string {
	if width <= 0 {
		return ""
	}
	if runewidth.StringWidth(status) <= width {
		return status
	}
	if width <= 3 {
		return strings.Repeat(".", width)
	}
	return runewidth.Truncate(status, width, "...")
}
