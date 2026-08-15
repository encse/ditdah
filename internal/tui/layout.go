package tui

import (
	"morsemanual/internal/tui/components"
	"morsemanual/internal/tui/keybinding"

	"github.com/rivo/tview"
)

// Layout arranges the shared application header and footer around replaceable
// page content.
type Layout interface {
	tview.Primitive
	Header() components.Header
	Footer() components.Footer
	Show(page Page)
}

type layout struct {
	*tview.Flex
	header       components.Header
	contentArea  *tview.Flex
	emptyContent components.TextView
	footer       components.Footer
}

// NewLayout creates the shared application layout from application-styled
// components.
func NewLayout(controls components.Factory) Layout {
	header := controls.Header()
	footer := controls.Footer()
	contentArea := tview.NewFlex()
	emptyContent := controls.TextView()

	layout := &layout{
		Flex:         tview.NewFlex().SetDirection(tview.FlexRow),
		header:       header,
		contentArea:  contentArea,
		emptyContent: emptyContent,
		footer:       footer,
	}
	layout.
		AddItem(header, 1, 0, false).
		AddItem(contentArea, 0, 1, true).
		AddItem(footer, 1, 0, false)
	layout.setContent(nil)
	return layout
}

func (l *layout) Header() components.Header {
	return l.header
}

func (l *layout) Footer() components.Footer {
	return l.footer
}

func (l *layout) Show(page Page) {
	l.header.SetTitle(page.Title())
	l.footer.SetKeyHints(keybinding.Hints(page.KeyBindings()))
	l.setContent(page.Content())
}

func (l *layout) setContent(content tview.Primitive) {
	if content == nil {
		content = l.emptyContent
	}
	l.contentArea.Clear().AddItem(content, 0, 1, true)
}
