package overlay

import (
	"testing"

	"github.com/rivo/tview"
)

func TestHostStacksAndRestoresFocus(t *testing.T) {
	app := tview.NewApplication()
	content := tview.NewBox()
	host := New(app)
	host.SetContent(content)
	changes := 0
	host.SetChangedFunc(func() {
		changes++
	})
	app.SetFocus(content)

	modal := tview.NewBox()
	modalHandle := host.Push(modal)
	if got := host.Top(); got != modal {
		t.Fatalf("top overlay = %T, want modal", got)
	}
	if changes != 1 {
		t.Fatalf("change count after modal push = %d, want 1", changes)
	}
	if got := app.GetFocus(); got != modal {
		t.Fatalf("focus after modal push = %T, want modal", got)
	}

	popup := tview.NewBox()
	popupHandle := host.Push(popup)
	if got := host.Top(); got != popup {
		t.Fatalf("top overlay = %T, want popup", got)
	}
	if got := app.GetFocus(); got != popup {
		t.Fatalf("focus after popup push = %T, want popup", got)
	}
	if got := host.FocusBeforeTop(); got != modal {
		t.Fatalf("focus before popup = %T, want modal", got)
	}

	popupHandle.Close()
	if changes != 3 {
		t.Fatalf("change count after popup close = %d, want 3", changes)
	}
	if got := app.GetFocus(); got != modal {
		t.Fatalf("focus after popup close = %T, want modal", got)
	}
	if !host.Active() {
		t.Fatal("host is inactive while modal is still open")
	}

	modalHandle.Close()
	if changes != 4 {
		t.Fatalf("change count after modal close = %d, want 4", changes)
	}
	if got := app.GetFocus(); got != content {
		t.Fatalf("focus after modal close = %T, want content", got)
	}
	if host.Active() {
		t.Fatal("host is active after closing all overlays")
	}
	if got := host.FocusBeforeTop(); got != nil {
		t.Fatalf("focus before top without overlay = %T, want nil", got)
	}
}
