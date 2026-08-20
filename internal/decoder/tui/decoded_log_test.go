package tui

import (
	"strings"
	"testing"

	"github.com/gdamore/tcell/v2"
)

func TestHighlightDecodedCallsignMatchesInsideWords(t *testing.T) {
	got := highlightDecodedCallsign("CQ deha7ncs HA7NCS/p", "ha7ncs")
	wantTag := strings.ToUpper(
		`["` + decodedCallsignRegion + `"]HA7NCS[""]`,
	)
	if strings.Count(strings.ToUpper(got), wantTag) != 2 {
		t.Fatalf("highlighted text = %q, want two embedded matches", got)
	}
}

func TestSelectedCallsignHighlightsExistingAndNewDecodedText(t *testing.T) {
	page := New(t.Context(), newTestHost(), nil, nil, nil, nil).(*page)
	if err := page.appendDecoded(t.Context(), "CQ DEHA7NCS "); err != nil {
		t.Fatal(err)
	}
	page.selectedCallsign = "HA7NCS"
	page.renderDecodedText()
	if !strings.Contains(page.output.Text(), decodedCallsignRegion) {
		t.Fatalf("existing decoded text = %q, want highlighted callsign", page.output.Text())
	}
	page.output.SetRect(0, 0, 24, 3)
	screen := tcell.NewSimulationScreen("UTF-8")
	if err := screen.Init(); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(screen.Fini)
	screen.SetSize(24, 3)
	page.output.Draw(screen)
	_, _, highlightedStyle, _ := screen.GetContent(6, 1)
	highlightedForeground, highlightedBackground, _ := highlightedStyle.Decompose()
	if highlightedForeground != tcell.ColorBlack ||
		highlightedBackground != tcell.ColorWhite {
		t.Fatalf(
			"highlight style = %v on %v, want black on white",
			highlightedForeground,
			highlightedBackground,
		)
	}

	if err := page.appendDecoded(t.Context(), "HA7NCS"); err != nil {
		t.Fatal(err)
	}
	if strings.Count(page.output.Text(), decodedCallsignRegion) != 2 {
		t.Fatalf("decoded text = %q, want two highlights", page.output.Text())
	}
}

func TestClearLogBindingRequiresConfirmation(t *testing.T) {
	host := newTestHost()
	page := New(t.Context(), host, nil, nil, nil, nil).(*page)
	if err := page.appendDecoded(t.Context(), "CQ CQ"); err != nil {
		t.Fatal(err)
	}
	binding := page.KeyBindings()[3]
	if !binding.Handle(tcell.NewEventKey(tcell.KeyRune, 'c', 0)) {
		t.Fatal("c clear binding was not handled")
	}
	dialog, ok := host.opened.(*clearLogDialog)
	if !ok {
		t.Fatalf("c opened %T, want *clearLogDialog", host.opened)
	}
	if page.decodedText.String() == "" {
		t.Fatal("opening confirmation cleared the log")
	}
	dialog.clear.InputHandler()(
		tcell.NewEventKey(tcell.KeyEnter, 0, 0),
		nil,
	)
	if page.decodedText.String() != "" || page.output.Text() != "" {
		t.Fatalf("cleared log = %q / %q", page.decodedText.String(), page.output.Text())
	}
}

func TestDecodedOutputFollowsOnlyWhileScrolledToEnd(t *testing.T) {
	page := New(t.Context(), newTestHost(), nil, nil, nil, nil).(*page)
	page.output.SetRect(0, 0, 20, 5)
	if err := page.appendDecoded(
		t.Context(),
		"one\ntwo\nthree\nfour\nfive\n",
	); err != nil {
		t.Fatal(err)
	}
	screen := tcell.NewSimulationScreen("UTF-8")
	if err := screen.Init(); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(screen.Fini)
	screen.SetSize(20, 5)
	page.output.Draw(screen)

	page.output.InputHandler()(tcell.NewEventKey(tcell.KeyUp, 0, 0), nil)
	page.output.Draw(screen)
	scrolledRow, _ := page.output.ScrollOffset()
	if err := page.appendDecoded(t.Context(), "six\n"); err != nil {
		t.Fatal(err)
	}
	page.output.Draw(screen)
	rowAfterAppend, _ := page.output.ScrollOffset()
	if rowAfterAppend != scrolledRow {
		t.Fatalf("scrolled row moved from %d to %d", scrolledRow, rowAfterAppend)
	}

	for range 10 {
		if page.output.AtEnd() {
			break
		}
		page.output.InputHandler()(tcell.NewEventKey(tcell.KeyDown, 0, 0), nil)
		page.output.Draw(screen)
	}
	bottomRow, _ := page.output.ScrollOffset()
	if !page.output.AtEnd() {
		t.Fatal("manual scroll down did not reach the end")
	}
	if err := page.appendDecoded(t.Context(), "seven\n"); err != nil {
		t.Fatal(err)
	}
	page.output.Draw(screen)
	followedRow, _ := page.output.ScrollOffset()
	if followedRow <= bottomRow {
		t.Fatalf("bottom row stayed at %d after append, got %d", bottomRow, followedRow)
	}
}
