package modal

import (
	"strings"
	"testing"

	"ditdah/internal/tui/components"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

func TestConfirmClosesBeforeCallingConfirmedAction(t *testing.T) {
	host := newDialogTestHost()
	confirmed := false
	dialog := OpenDangerConfirm(
		host,
		newDialogTestOwner(),
		" Confirm ",
		"Continue?",
		"This cannot be undone.",
		"Continue",
		func() {
			confirmed = true
			if !host.handle.closed {
				t.Error("confirmation callback ran before dialog closed")
			}
		},
	)

	pressDialogControl(t, dialog, 1)

	if !confirmed {
		t.Fatal("confirmation callback did not run")
	}
}

func TestConfirmCancelOnlyCloses(t *testing.T) {
	host := newDialogTestHost()
	confirmed := false
	dialog := OpenConfirm(
		host,
		newDialogTestOwner(),
		" Confirm ",
		"Continue?",
		"",
		"Continue",
		func() { confirmed = true },
	)

	pressDialogControl(t, dialog, 0)

	if !host.handle.closed {
		t.Fatal("Cancel did not close dialog")
	}
	if confirmed {
		t.Fatal("Cancel ran confirmation callback")
	}
}

func TestConfirmLeavesOneRowAboveActions(t *testing.T) {
	host := newDialogTestHost()
	dialog := OpenConfirm(
		host,
		newDialogTestOwner(),
		" Confirm ",
		"Synchronize pending QSOs with QRZ.com?",
		"",
		"Synchronize",
		func() {},
	)

	if got := dialog.Size(); got != (Size{Width: 58, Height: 7}) {
		t.Fatalf("dialog size = %#v, want 58x7", got)
	}
}

func TestErrorMessageClosesWithOK(t *testing.T) {
	host := newDialogTestHost()
	dialog := OpenError(
		host,
		newDialogTestOwner(),
		" Error ",
		"Error: disk failed",
	)
	if got := dialog.Size(); got != (Size{Width: 58, Height: 6}) {
		t.Fatalf("dialog size = %#v, want 58x6", got)
	}

	pressDialogControl(t, dialog, 0)

	if !host.handle.closed {
		t.Fatal("OK did not close error message")
	}
}

func TestErrorMessageGrowsToFitWrappedText(t *testing.T) {
	host := newDialogTestHost()
	dialog := OpenError(
		host,
		newDialogTestOwner(),
		" Error ",
		"Error: configure a QRZ.com Logbook API key in Settings first",
	)

	if got := dialog.Size(); got != (Size{Width: 58, Height: 7}) {
		t.Fatalf("dialog size = %#v, want 58x7", got)
	}
	screen := tcell.NewSimulationScreen("UTF-8")
	if err := screen.Init(); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(screen.Fini)
	screen.SetSize(58, 7)
	dialog.Content().SetRect(0, 0, 58, 7)
	dialog.Content().Draw(screen)

	var rendered strings.Builder
	for y := 0; y < 7; y++ {
		for x := 0; x < 58; x++ {
			character, _, _, _ := screen.GetContent(x, y)
			rendered.WriteRune(character)
		}
	}
	if !strings.Contains(rendered.String(), "Settings first") {
		t.Fatalf("wrapped error message was truncated: %q", rendered.String())
	}
}

type dialogTestHost struct {
	controls components.Factory
	handle   *dialogTestHandle
}

func newDialogTestHost() *dialogTestHost {
	return &dialogTestHost{controls: components.New(components.Dependencies{
		Theme: components.Theme{
			Background:       tcell.ColorBlack,
			PrimaryText:      tcell.ColorWhite,
			ButtonText:       tcell.ColorWhite,
			ButtonBackground: tcell.ColorBlue,
		},
		ModalTheme: components.Theme{
			Background:             tcell.ColorSilver,
			PrimaryText:            tcell.ColorBlack,
			MutedText:              tcell.ColorGray,
			DangerText:             tcell.ColorRed,
			ButtonText:             tcell.ColorBlack,
			ButtonBackground:       tcell.ColorBlue,
			DangerButtonBackground: tcell.ColorMaroon,
		},
	})}
}

func (h *dialogTestHost) Components() components.Factory { return h.controls }

func (h *dialogTestHost) OpenModal(
	_ Owner,
	_ Dialog,
) Handle {
	h.handle = &dialogTestHandle{}
	return h.handle
}

type dialogTestOwner struct{ *tview.Box }

func newDialogTestOwner() *dialogTestOwner {
	return &dialogTestOwner{Box: tview.NewBox()}
}

func (o *dialogTestOwner) Content() tview.Primitive { return o.Box }

type dialogTestHandle struct{ closed bool }

func (h *dialogTestHandle) Close() { h.closed = true }

func pressDialogControl(t *testing.T, dialog Dialog, index int) {
	t.Helper()
	focusables := dialog.Focusables()
	if index < 0 || index >= len(focusables) {
		t.Fatalf("dialog control index %d outside %d focusables", index, len(focusables))
	}
	focusables[index].InputHandler()(
		tcell.NewEventKey(tcell.KeyEnter, 0, 0),
		nil,
	)
}
