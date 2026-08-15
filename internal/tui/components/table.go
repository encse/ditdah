package components

import (
	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

// TableCellStyle selects one of the application table text styles.
type TableCellStyle int

const (
	TableCellBody TableCellStyle = iota
	TableCellHeader
	TableCellMuted
)

// TableCell describes a table cell without exposing tview's pointer model.
type TableCell struct {
	Text      string
	Style     TableCellStyle
	Disabled  bool
	MaxWidth  int
	Expansion int
}

// Table is an application-styled, row-selectable table.
type Table interface {
	tview.Primitive
	Clear()
	SetCell(row int, column int, cell TableCell)
	Select(row int, column int)
	Selection() (row int, column int)
	SetSelectionChangedFunc(handler func(row int, column int))
}

func (t *table) MouseHandler() mouseHandler {
	return keepMouseOwner(t, t.Table.MouseHandler())
}

type table struct {
	*tview.Table
	theme Theme
}

func newTable(title string, theme Theme, focusChanged func()) Table {
	view := tview.NewTable().
		SetBorders(false).
		SetSelectable(true, false).
		SetFixed(1, 0).
		SetSeparator(' ')
	view.SetBackgroundColor(theme.Background)
	view.SetBorder(true).
		SetBorderColor(theme.Border).
		SetTitle(title).
		SetTitleColor(theme.Accent)
	view.SetSelectedStyle(
		tcell.StyleDefault.
			Foreground(theme.SelectionText).
			Background(theme.SelectionBackground).
			Bold(true),
	)
	view.SetFocusFunc(func() {
		notify(focusChanged)
	})
	return &table{Table: view, theme: theme}
}

func (t *table) Clear() {
	t.Table.Clear()
}

func (t *table) SetCell(row int, column int, cell TableCell) {
	viewCell := tview.NewTableCell(cell.Text).
		SetTextColor(t.cellColor(cell.Style)).
		SetSelectable(!cell.Disabled).
		SetMaxWidth(cell.MaxWidth).
		SetExpansion(cell.Expansion)
	if cell.Style == TableCellHeader {
		viewCell.SetAttributes(tcell.AttrBold)
	}
	t.Table.SetCell(row, column, viewCell)
}

func (t *table) Select(row int, column int) {
	t.Table.Select(row, column)
}

func (t *table) Selection() (int, int) {
	return t.Table.GetSelection()
}

func (t *table) SetSelectionChangedFunc(handler func(row int, column int)) {
	t.Table.SetSelectionChangedFunc(handler)
}

func (t *table) cellColor(style TableCellStyle) tcell.Color {
	switch style {
	case TableCellHeader:
		return t.theme.PrimaryText
	case TableCellMuted:
		return t.theme.MutedText
	default:
		return t.theme.SecondaryText
	}
}
