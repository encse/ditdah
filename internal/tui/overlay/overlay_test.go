package overlay

import (
	"testing"

	"github.com/rivo/tview"
)

func TestHostStacksAndRestoresFocus(t *testing.T) {
	app := tview.NewApplication()
	content := tview.NewBox()
	host := New(app, content)
	app.SetFocus(content)

	modal := tview.NewBox()
	modalHandle := host.Push(modal)
	if got := app.GetFocus(); got != modal {
		t.Fatalf("focus after modal push = %T, want modal", got)
	}

	popup := tview.NewBox()
	popupHandle := host.Push(popup)
	if got := app.GetFocus(); got != popup {
		t.Fatalf("focus after popup push = %T, want popup", got)
	}

	popupHandle.Close()
	if got := app.GetFocus(); got != modal {
		t.Fatalf("focus after popup close = %T, want modal", got)
	}
	if !host.Active() {
		t.Fatal("host is inactive while modal is still open")
	}

	modalHandle.Close()
	if got := app.GetFocus(); got != content {
		t.Fatalf("focus after modal close = %T, want content", got)
	}
	if host.Active() {
		t.Fatal("host is active after closing all overlays")
	}
}
