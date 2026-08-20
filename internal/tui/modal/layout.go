package modal

import (
	"morsemanual/internal/tui/components"

	"github.com/rivo/tview"
)

const frameHeight = 2

type layoutRow struct {
	primitive tview.Primitive
	height    int
}

// Rows is a borderless, fixed-height row declaration used by paged layouts.
type Rows struct {
	content tview.Primitive
	height  int
}

type RowsBuilder struct {
	controls components.Factory
	rows     []layoutRow
}

func NewRows(controls components.Factory) *RowsBuilder {
	return &RowsBuilder{controls: controls.Modal()}
}

func (b *RowsBuilder) Gap(height int) *RowsBuilder {
	return b.Row(nil, height)
}

func (b *RowsBuilder) Row(
	primitive tview.Primitive,
	height int,
) *RowsBuilder {
	if height > 0 {
		b.rows = append(b.rows, layoutRow{primitive: primitive, height: height})
	}
	return b
}

func (b *RowsBuilder) Actions(actions tview.Primitive) Rows {
	b.Row(actions, 1)
	return b.Build()
}

func (b *RowsBuilder) Build() Rows {
	content := b.controls.Flex(tview.FlexRow)
	height := 0
	for _, row := range b.rows {
		content.AddItem(row.primitive, row.height, 0, false)
		height += row.height
	}
	return Rows{content: content, height: height}
}

type Page struct {
	Name    string
	Rows    Rows
	Visible bool
}

// Layout is a modal surface whose size is derived from its declared rows.
// Embedding it gives a dialog consistent Content and Size implementations.
type Layout struct {
	content tview.Primitive
	size    Size
}

func (l Layout) Content() tview.Primitive { return l.content }

func (l Layout) Size() Size { return l.size }

// LayoutBuilder declares the rows inside one consistently themed modal frame.
type LayoutBuilder struct {
	controls          components.Factory
	title             string
	width             int
	horizontalPadding int
	rows              []layoutRow
}

// NewLayout starts a fixed-row modal layout. Its height is calculated by
// Build or Actions; callers never specify it independently from the content.
func NewLayout(
	controls components.Factory,
	title string,
	width int,
) *LayoutBuilder {
	return &LayoutBuilder{
		controls: controls.Modal(),
		title:    title,
		width:    width,
	}
}

func (b *LayoutBuilder) Padding(columns int) *LayoutBuilder {
	if columns < 0 {
		columns = 0
	}
	b.horizontalPadding = columns
	return b
}

func (b *LayoutBuilder) Gap(height int) *LayoutBuilder {
	return b.Row(nil, height)
}

func (b *LayoutBuilder) Row(
	primitive tview.Primitive,
	height int,
) *LayoutBuilder {
	if height > 0 {
		b.rows = append(b.rows, layoutRow{
			primitive: primitive,
			height:    height,
		})
	}
	return b
}

// Actions adds the terminal button row and finishes the layout. Because it
// returns Layout instead of LayoutBuilder, rows cannot be appended below it.
func (b *LayoutBuilder) Actions(actions tview.Primitive) Layout {
	b.Row(actions, 1)
	return b.build()
}

func (b *LayoutBuilder) build() Layout {
	body := b.controls.Flex(tview.FlexRow)
	height := frameHeight
	for _, row := range b.rows {
		body.AddItem(row.primitive, row.height, 0, false)
		height += row.height
	}
	content := tview.Primitive(body)
	if b.horizontalPadding > 0 {
		content = b.controls.Flex(tview.FlexColumn).
			AddItem(nil, b.horizontalPadding, 0, false).
			AddItem(body, 0, 1, false).
			AddItem(nil, b.horizontalPadding, 0, false)
	}
	stack := b.controls.PageStack(b.title)
	stack.Add("content", content, true)
	return Layout{
		content: stack,
		size: Size{
			Width:  b.width,
			Height: height,
		},
	}
}

// NewPagedLayout creates one modal frame shared by multiple row-declared
// states. Its height is derived from the tallest page.
func NewPagedLayout(
	controls components.Factory,
	title string,
	width int,
	horizontalPadding int,
	pageDefinitions ...Page,
) (Layout, components.PageStack) {
	controls = controls.Modal()
	pages := controls.PageStack(title)
	maxHeight := 0
	for _, definition := range pageDefinitions {
		content := definition.Rows.content
		if horizontalPadding > 0 {
			content = controls.Flex(tview.FlexColumn).
				AddItem(nil, horizontalPadding, 0, false).
				AddItem(content, 0, 1, false).
				AddItem(nil, horizontalPadding, 0, false)
		}
		pages.Add(definition.Name, content, definition.Visible)
		if definition.Rows.height > maxHeight {
			maxHeight = definition.Rows.height
		}
	}
	return Layout{
		content: pages,
		size:    Size{Width: width, Height: frameHeight + maxHeight},
	}, pages
}
