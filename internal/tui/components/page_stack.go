package components

import (
	"github.com/rivo/tview"
)

// PageStack displays one of multiple interchangeable primitives inside one
// consistently styled surface.
type PageStack interface {
	tview.Primitive
	Add(name string, primitive tview.Primitive, visible bool)
	Show(name string)
	Active() string
}

type pageStack struct {
	*tview.Pages
}

func newPageStack(
	title string,
	theme Theme,
	focusChanged func(),
) PageStack {
	pages := tview.NewPages()
	pages.SetBackgroundColor(theme.Background)
	pages.SetBorder(true).
		SetBorderColor(theme.Border).
		SetTitle(title).
		SetTitleColor(theme.Accent)
	pages.SetFocusFunc(func() { notify(focusChanged) })
	return &pageStack{Pages: pages}
}

func (p *pageStack) Add(
	name string,
	primitive tview.Primitive,
	visible bool,
) {
	p.AddPage(name, primitive, true, visible)
}

func (p *pageStack) Show(name string) {
	p.SwitchToPage(name)
}

func (p *pageStack) Active() string {
	name, _ := p.GetFrontPage()
	return name
}
