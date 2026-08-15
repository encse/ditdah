// Package overlay provides stackable overlay layers for the terminal UI.
package overlay

import (
	"fmt"

	"github.com/rivo/tview"
	"morsemanual/internal/tui/components"
)

// Host displays component overlays and exposes their shared tview root.
type Host interface {
	components.OverlayHost
	Root() tview.Primitive
	SetContent(content tview.Primitive)
	SetChangedFunc(handler func())
	Active() bool
}

type host struct {
	app     *tview.Application
	pages   *tview.Pages
	nextID  uint64
	entries []entry
	changed func()
}

type entry struct {
	name          string
	previousFocus tview.Primitive
}

type handle struct {
	host   *host
	name   string
	closed bool
}

// New creates an initially empty overlay host.
func New(app *tview.Application) Host {
	return &host{
		app:   app,
		pages: tview.NewPages(),
	}
}

func (h *host) Root() tview.Primitive {
	return h.pages
}

func (h *host) SetContent(content tview.Primitive) {
	h.pages.RemovePage("content").
		AddPage("content", content, true, true)
}

func (h *host) SetChangedFunc(handler func()) {
	h.changed = handler
}

func (h *host) Active() bool {
	return len(h.entries) > 0
}

func (h *host) Push(primitive tview.Primitive) components.Overlay {
	h.nextID++
	name := fmt.Sprintf("overlay-%d", h.nextID)
	h.entries = append(h.entries, entry{
		name:          name,
		previousFocus: h.app.GetFocus(),
	})
	h.pages.AddPage(name, primitive, true, true)
	h.app.SetFocus(primitive)
	h.notifyChanged()
	return &handle{host: h, name: name}
}

func (h *host) close(name string) {
	index := -1
	for entryIndex, entry := range h.entries {
		if entry.name == name {
			index = entryIndex
			break
		}
	}
	if index < 0 {
		return
	}

	previousFocus := h.entries[index].previousFocus
	for entryIndex := len(h.entries) - 1; entryIndex >= index; entryIndex-- {
		h.pages.RemovePage(h.entries[entryIndex].name)
	}
	h.entries = h.entries[:index]
	h.app.SetFocus(previousFocus)
	h.notifyChanged()
}

func (h *host) notifyChanged() {
	if h.changed != nil {
		h.changed()
	}
}

func (h *handle) Close() {
	if h.closed {
		return
	}
	h.closed = true
	h.host.close(h.name)
}
