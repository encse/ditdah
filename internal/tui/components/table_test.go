package components

import (
	"reflect"
	"testing"

	"morsemanual/internal/tui/keybinding"
)

func TestTableUsesValueCellsAndTheme(t *testing.T) {
	screen := newTestScreen(t)
	table := newTestFactory().Table(" Test ")
	table.SetCell(0, 0, TableCell{
		Text:     "Heading",
		Style:    TableCellHeader,
		Disabled: true,
	})
	table.SetCell(1, 0, TableCell{Text: "Value"})
	table.SetRect(1, 1, 30, 6)
	table.Focus(nil)
	table.Select(1, 0)
	table.Draw(screen)

	row, column := table.Selection()
	if row != 1 || column != 0 {
		t.Fatalf("Selection() = (%d, %d), want (1, 0)", row, column)
	}
	assertForeground(t, screen, 2, 2, testTheme().PrimaryText)
	assertBackground(t, screen, 2, 3, testTheme().SelectionBackground)
}

func TestTablePublishesNativeKeyHints(t *testing.T) {
	table := newTestFactory().Table(" Test ")
	want := []keybinding.Hint{
		{Keys: "↑/k ↓/j", Description: "move"},
		{Keys: "←/h →/l", Description: "scroll"},
		{Keys: "PgUp/PgDn", Description: "page"},
		{Keys: "Home/End g/G", Description: "first/last"},
	}

	if got := table.KeyHints(); !reflect.DeepEqual(got, want) {
		t.Fatalf("KeyHints() = %#v, want %#v", got, want)
	}
}
