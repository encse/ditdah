package components

import (
	"strings"
	"testing"

	"github.com/rivo/tview"
)

func TestHeaderKeepsMenuSpacerAndStatusSeparate(t *testing.T) {
	screen := newTestScreen(t)
	header := newTestFactory().Header()
	menu := newTestFactory().TextView()
	menu.SetText("Menu")
	menu.SetTextAlign(tview.AlignCenter)
	header.SetMenu(menu, 12)
	header.SetStatus("ready")
	header.SetRect(0, 0, 40, 1)
	header.Draw(screen)

	assertRune(t, screen, 4, 0, 'M')
	assertRune(t, screen, 20, 0, ' ')
	assertRune(t, screen, 35, 0, 'r')
	assertForeground(t, screen, 35, 0, testTheme().MutedText)
}

func TestHeaderTruncatesLongStatusWithoutCoveringMenu(t *testing.T) {
	header := newTestFactory().Header().(*header)
	menu := newTestFactory().TextView()
	menu.SetText("Menu")
	header.SetMenu(menu, 12)
	header.SetRect(0, 0, 40, 1)
	header.SetStatus(strings.Repeat("audio error ", 20))

	if got := tview.TaggedStringWidth(header.status.Text()); got != 27 {
		t.Fatalf("status width = %d, want available width 27", got)
	}
	if !strings.HasSuffix(header.status.Text(), "...") {
		t.Fatalf("truncated status = %q, want ellipsis", header.status.Text())
	}
}

func TestHeaderCapsStatusOnWideScreens(t *testing.T) {
	header := newTestFactory().Header().(*header)
	header.SetRect(0, 0, 160, 1)
	header.SetStatus(strings.Repeat("x", 100))

	if got := tview.TaggedStringWidth(header.status.Text()); got != maxHeaderStatusWidth {
		t.Fatalf("status width = %d, want maximum %d", got, maxHeaderStatusWidth)
	}
	if !strings.HasSuffix(header.status.Text(), "...") {
		t.Fatalf("truncated status = %q, want ellipsis", header.status.Text())
	}
}

func TestHeaderCanReplaceMenu(t *testing.T) {
	header := newTestFactory().Header().(*header)
	first := newTestFactory().TextView()
	first.SetText("first")
	second := newTestFactory().TextView()
	second.SetText("second")

	header.SetMenu(first, 10)
	header.SetMenu(second, 10)

	if header.menu != second {
		t.Fatalf("menu = %T, want replacement menu", header.menu)
	}
}
