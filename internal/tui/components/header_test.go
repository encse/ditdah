package components

import (
	"testing"

	"github.com/rivo/tview"
)

func TestHeaderKeepsTitleMenuAndStatusSeparate(t *testing.T) {
	screen := newTestScreen(t)
	header := newTestFactory().Header()
	menu := newTestFactory().TextView()
	menu.SetText("Menu")
	menu.SetTextAlign(tview.AlignCenter)
	header.SetTitle("Logbook")
	header.SetMenu(menu, 12)
	header.SetStatus("ready")
	header.SetRect(0, 0, 40, 1)
	header.Draw(screen)

	assertRune(t, screen, 4, 0, 'M')
	assertRune(t, screen, 16, 0, 'L')
	assertForeground(t, screen, 16, 0, testTheme().Accent)
	assertRune(t, screen, 35, 0, 'r')
	assertForeground(t, screen, 35, 0, testTheme().MutedText)
}

func TestHeaderCanReplaceMenu(t *testing.T) {
	header := newTestFactory().Header().(*header)
	first := newTestFactory().TextView()
	first.SetText("first")
	second := newTestFactory().TextView()
	second.SetText("second")

	header.SetMenu(first, 10)
	header.SetMenu(second, 10)

	if got := header.GetItemCount(); got != 3 {
		t.Fatalf("item count = %d, want 3", got)
	}
	if got := header.GetItem(0); got != second {
		t.Fatalf("menu = %T, want replacement menu", got)
	}
}
