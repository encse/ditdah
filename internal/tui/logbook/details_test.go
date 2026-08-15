package logbook

import (
	"strings"
	"testing"

	"github.com/gdamore/tcell/v2"
)

func TestDetailsKeepWrappedColumnsAligned(t *testing.T) {
	page, _ := newTestPage(t)
	page.details.setRows([]detailRow{{
		left: detailField{
			label: "Name",
			value: strings.Repeat("Long value ", 10),
		},
		right: detailField{label: "QTH", value: "Constanta"},
	}})
	screen := newTestScreen(t, 80, 12)
	page.details.SetRect(0, 0, 80, 8)
	page.details.Draw(screen)

	assertRune(t, screen, 42, 1, 'Q')
}

func TestDetailsScrollAsOneView(t *testing.T) {
	page, _ := newTestPage(t)
	page.details.setRows([]detailRow{{
		left: detailField{
			label: "Notes",
			value: strings.Repeat("wrapped content ", 20),
		},
		right: detailField{label: "QTH", value: "Budapest"},
	}})
	screen := newTestScreen(t, 40, 12)
	page.details.SetRect(0, 0, 40, 6)
	page.details.Draw(screen)
	page.details.Focus(nil)
	page.details.InputHandler()(tcell.NewEventKey(tcell.KeyDown, 0, 0), nil)

	row, _ := page.details.ScrollOffset()
	if row != 1 {
		t.Fatalf("scroll row = %d, want 1", row)
	}
}
