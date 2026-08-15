package tui

import (
	"testing"

	"github.com/gdamore/tcell/v2"
)

func TestModalThemeUsesRequestedBackgroundAndWhiteText(t *testing.T) {
	theme := nordTheme.modalComponents()
	wantBackground := tcell.NewRGBColor(190, 190, 190)
	if theme.Background != wantBackground {
		t.Fatalf("modal background = %v, want %v", theme.Background, wantBackground)
	}
	if theme.PrimaryText != tcell.ColorWhite || theme.Accent != tcell.ColorWhite {
		t.Fatalf(
			"modal text colors = primary %v, accent %v, want white",
			theme.PrimaryText,
			theme.Accent,
		)
	}
}
