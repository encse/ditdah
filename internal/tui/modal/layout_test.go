package modal

import (
	"testing"

	"morsemanual/internal/tui/components"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

func TestLayoutDerivesHeightAndEndsAtActions(t *testing.T) {
	modalBackground := tcell.ColorSilver
	controls := components.New(components.Dependencies{
		Theme: components.Theme{Background: tcell.ColorBlack},
		ModalTheme: components.Theme{
			Background:       modalBackground,
			PrimaryText:      tcell.ColorBlack,
			Border:           tcell.ColorWhite,
			Accent:           tcell.ColorWhite,
			ButtonText:       tcell.ColorBlack,
			ButtonBackground: tcell.ColorBlue,
		},
	}).Modal()
	message := controls.TextView()
	message.SetText("Confirm")
	actions := controls.Flex(tview.FlexColumn).
		AddItem(controls.Button("Cancel"), 10, 0, false).
		AddItem(controls.Button("OK"), 10, 0, false)
	layout := NewLayout(controls, " Confirm ", 32).
		Padding(2).
		Gap(1).
		Row(message, 1).
		Actions(actions)

	if got := layout.Size(); got != (Size{Width: 32, Height: 5}) {
		t.Fatalf("layout size = %#v, want 32x5", got)
	}
	screen := tcell.NewSimulationScreen("UTF-8")
	if err := screen.Init(); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(screen.Fini)
	screen.SetSize(32, 5)
	layout.Content().SetRect(0, 0, 32, 5)
	layout.Content().Draw(screen)
	if character, _, _, _ := screen.GetContent(3, 4); character != tview.Borders.Horizontal {
		t.Fatalf("row below actions = %q, want bottom border", character)
	}
	for y := 1; y < 4; y++ {
		_, _, style, _ := screen.GetContent(2, y)
		_, background, _ := style.Decompose()
		if background != modalBackground {
			t.Fatalf("background at row %d = %v, want %v", y, background, modalBackground)
		}
	}
}
