package modal

import (
	"testing"

	"morsemanual/internal/tui/components"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

func TestConfirmClosesBeforeCallingConfirmedAction(t *testing.T) {
	host := newDialogTestHost()
	confirmed := false
	dialog := OpenDangerConfirm(
		host,
		tview.NewBox(),
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
		tview.NewBox(),
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

func TestErrorMessageClosesWithOK(t *testing.T) {
	host := newDialogTestHost()
	dialog := OpenError(
		host,
		tview.NewBox(),
		" Error ",
		"Error: disk failed",
	)

	pressDialogControl(t, dialog, 0)

	if !host.handle.closed {
		t.Fatal("OK did not close error message")
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
	_ tview.Primitive,
	_ Dialog,
) Handle {
	h.handle = &dialogTestHandle{}
	return h.handle
}

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
