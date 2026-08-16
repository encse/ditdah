package tui

import (
	"testing"

	"morsemanual/internal/tui/components"
	"morsemanual/internal/tui/keybinding"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

func TestLayoutArrangesHeaderContentAndFooter(t *testing.T) {
	screen := newLayoutTestScreen(t)
	controls := components.New(components.Dependencies{
		Theme:      nordTheme.components(),
		ModalTheme: nordTheme.modalComponents(),
	})
	layout := NewLayout(controls)
	content := controls.TextView()
	content.SetText("content")
	layout.Footer().SetContext("24 QSOs")
	layout.Show(testPage{
		id:      "logbook",
		title:   "Logbook",
		content: content,
	})
	layout.SetRect(0, 0, 40, 12)
	layout.Draw(screen)

	assertLayoutRune(t, screen, 16, 0, 'L')
	_, _, headerStyle, _ := screen.GetContent(30, 0)
	_, headerBackground, _ := headerStyle.Decompose()
	if want := tcell.NewRGBColor(190, 190, 190); headerBackground != want {
		t.Fatalf("header background = %v, want %v", headerBackground, want)
	}
	assertLayoutRune(t, screen, 0, 1, 'c')
	assertLayoutRune(t, screen, 0, 11, '2')
}

func TestLayoutReplacesContent(t *testing.T) {
	controls := components.New(components.Dependencies{
		Theme: nordTheme.components(),
	})
	layout := NewLayout(controls).(*layout)
	first := controls.TextView()
	second := controls.TextView()

	layout.Show(testPage{id: "first", title: "First", content: first})
	layout.Show(testPage{id: "second", title: "Second", content: second})

	if got := layout.contentArea.GetItemCount(); got != 1 {
		t.Fatalf("content item count = %d, want 1", got)
	}
	if got := layout.contentArea.GetItem(0); got != second {
		t.Fatalf("content = %T, want replacement content", got)
	}
}

type testPage struct {
	id      string
	title   string
	content tview.Primitive
}

func (p testPage) ID() string {
	return p.id
}

func (p testPage) Title() string {
	return p.title
}

func (p testPage) Content() tview.Primitive {
	return p.content
}

func (p testPage) Focusables() []tview.Primitive {
	return nil
}

func (p testPage) KeyBindings() []keybinding.Binding {
	return nil
}

func newLayoutTestScreen(t *testing.T) tcell.SimulationScreen {
	t.Helper()
	screen := tcell.NewSimulationScreen("UTF-8")
	if err := screen.Init(); err != nil {
		t.Fatalf("initialize screen: %v", err)
	}
	t.Cleanup(screen.Fini)
	screen.SetSize(40, 12)
	return screen
}

func assertLayoutRune(
	t *testing.T,
	screen tcell.Screen,
	x int,
	y int,
	want rune,
) {
	t.Helper()
	got, _, _, _ := screen.GetContent(x, y)
	if got != want {
		t.Fatalf("rune at (%d, %d) = %q, want %q", x, y, got, want)
	}
}
