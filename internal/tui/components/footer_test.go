package components

import (
	"testing"

	"ditdah/internal/tui/keybinding"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

func TestFooterKeepsContextAndKeyHintsSeparate(t *testing.T) {
	footer := newTestFactory().Footer().(*footer)
	footer.SetContext("24 QSOs")
	footer.SetKeyBindings([]keybinding.Binding{
		keybinding.On("move", func() {}, keybinding.Key(tcell.KeyUp), keybinding.Key(tcell.KeyDown)),
		keybinding.OnRune('q', "quit", func() {}),
	})

	if got := footer.context.Text(); got != "24 QSOs" {
		t.Fatalf("context = %q, want %q", got, "24 QSOs")
	}
	wantHints := "Up/Down move   q quit"
	if got := footer.hints.Text(); got != wantHints {
		t.Fatalf("hints = %q, want %q", got, wantHints)
	}
}

func TestFooterGivesHintsFullWidthWithoutContext(t *testing.T) {
	footer := newTestFactory().Footer().(*footer)
	if got := footer.GetItemCount(); got != 1 {
		t.Fatalf("item count without context = %d, want 1", got)
	}
	if got := footer.GetItem(0); got != footer.hints {
		t.Fatalf("only item without context = %T, want hints", got)
	}

	footer.SetContext("24 QSOs")
	if got := footer.GetItemCount(); got != 2 {
		t.Fatalf("item count with context = %d, want 2", got)
	}

	footer.SetContext("")
	if got := footer.GetItemCount(); got != 1 {
		t.Fatalf("item count after clearing context = %d, want 1", got)
	}
	if got := footer.GetItem(0); got != footer.hints {
		t.Fatalf("only item after clearing context = %T, want hints", got)
	}
}

func TestFooterDisplaysKeyHintMarkupLiterally(t *testing.T) {
	footer := newTestFactory().Footer().(*footer)
	footer.SetKeyBindings([]keybinding.Binding{
		keybinding.OnRune('[', "use [brackets]", func() {}),
	})

	want := "[ use [brackets]"
	if got := footer.hints.Text(); got != want {
		t.Fatalf("hints = %q, want %q", got, want)
	}
}

func TestFooterKeyHintsAreClickable(t *testing.T) {
	invoked := 0
	footer := newTestFactory().Footer().(*footer)
	footer.SetKeyBindings([]keybinding.Binding{
		keybinding.OnRune('q', "quit", func() { invoked++ }),
	})
	footer.SetRect(0, 0, 20, 1)

	consumed, _ := footer.MouseHandler()(
		tview.MouseLeftClick,
		tcell.NewEventMouse(7, 0, tcell.ButtonNone, 0),
		func(tview.Primitive) {},
	)
	if !consumed || invoked != 1 {
		t.Fatalf("click consumed = %v, invocations = %d; want true, 1", consumed, invoked)
	}
}
