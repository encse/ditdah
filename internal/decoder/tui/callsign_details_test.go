package tui

import (
	"strings"
	"testing"

	"ditdah/internal/callsign"
	"ditdah/internal/optional"

	"github.com/gdamore/tcell/v2"
)

func TestCallsignDetailsAlignLabelsAndValues(t *testing.T) {
	view := newCallsignDetailsView(newTestHost().Components())
	view.setEntry(callsign.Entry{
		Status: callsign.StatusReady,
		Record: optional.Some(callsign.Record{
			Callsign: "HA5LA",
			Name:     optional.Some("Laszlo"),
			Nickname: optional.Some("Laci"),
			Grid:     optional.Some("JN97mm"),
		}),
	})

	screen := newCallsignDetailsScreen(t, 48, 8)
	view.SetRect(0, 0, 48, 8)
	view.Draw(screen)

	assertCallsignDetailsRune(t, screen, 1, 1, 'C')
	assertCallsignDetailsRune(t, screen, 11, 1, 'H')
	assertCallsignDetailsRune(t, screen, 11, 2, 'L')
	assertCallsignDetailsRune(t, screen, 11, 3, 'L')
	assertCallsignDetailsRune(t, screen, 11, 4, 'J')
}

func TestCallsignDetailsIndentWrappedValues(t *testing.T) {
	view := newCallsignDetailsView(newTestHost().Components())
	view.fields = []callsignDetailField{{
		label: "Name",
		value: strings.Repeat("Long value ", 8),
	}}
	view.renderedWidth = -1

	screen := newCallsignDetailsScreen(t, 24, 8)
	view.SetRect(0, 0, 24, 8)
	view.Draw(screen)

	assertCallsignDetailsRune(t, screen, 11, 1, 'L')
	assertCallsignDetailsRune(t, screen, 11, 2, 'L')
}

func newCallsignDetailsScreen(
	t *testing.T,
	width int,
	height int,
) tcell.SimulationScreen {
	t.Helper()
	screen := tcell.NewSimulationScreen("UTF-8")
	if err := screen.Init(); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(screen.Fini)
	screen.SetSize(width, height)
	return screen
}

func assertCallsignDetailsRune(
	t *testing.T,
	screen tcell.SimulationScreen,
	x int,
	y int,
	want rune,
) {
	t.Helper()
	got, _, _, _ := screen.GetContent(x, y)
	if got != want {
		t.Fatalf("screen cell (%d, %d) = %q, want %q", x, y, got, want)
	}
}
