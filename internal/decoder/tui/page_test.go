package tui

import (
	"testing"

	"morsemanual/internal/tui/components"
	"morsemanual/internal/tui/modal"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

func TestPageMetadata(t *testing.T) {
	page := New(newTestHost())

	if page.ID() != "morse-decoder" {
		t.Fatalf("ID() = %q, want morse-decoder", page.ID())
	}
	if page.Title() != "Morse decoder" {
		t.Fatalf("Title() = %q, want Morse decoder", page.Title())
	}
	if page.Content() == nil {
		t.Fatal("Content() is nil")
	}
	if len(page.Focusables()) != 0 {
		t.Fatalf("Focusables() = %d items, want none", len(page.Focusables()))
	}
}

func TestPageShowsDecoderOutputPlaceholder(t *testing.T) {
	page := New(newTestHost()).(*page)
	if page.output.Text() != "Decoder output will appear here." {
		t.Fatalf("output text = %q", page.output.Text())
	}
}

type testHost struct {
	controls components.Factory
}

func newTestHost() testHost {
	theme := components.Theme{
		Background:     tcell.ColorBlack,
		PrimaryText:    tcell.ColorWhite,
		MutedText:      tcell.ColorGray,
		FieldTextColor: tcell.ColorWhite,
	}
	return testHost{controls: components.New(components.Dependencies{Theme: theme})}
}

func (h testHost) SetFocus(tview.Primitive) {}

func (h testHost) Refresh() {}

func (h testHost) Components() components.Factory { return h.controls }

func (h testHost) OpenModal(modal.Dialog) modal.Handle { return nil }
