package modal

import (
	"ditdah/internal/tui/components"
	"ditdah/internal/tui/keybinding"

	"github.com/rivo/tview"
)

type Opener interface {
	Components() components.Factory
	OpenModal(owner Owner, dialog Dialog) Handle
}

type confirmDialog struct {
	Layout
	confirm       components.Button
	cancel        components.Button
	confirmAction func()
	handle        Handle
}

func OpenConfirm(
	host Opener,
	owner Owner,
	title string,
	message string,
	detail string,
	confirmLabel string,
	confirmAction func(),
) Dialog {
	return openConfirm(
		host, owner, title, message, detail, confirmLabel, confirmAction, false,
	)
}

func OpenDangerConfirm(
	host Opener,
	owner Owner,
	title string,
	message string,
	detail string,
	confirmLabel string,
	confirmAction func(),
) Dialog {
	return openConfirm(
		host, owner, title, message, detail, confirmLabel, confirmAction, true,
	)
}

func openConfirm(
	host Opener,
	owner Owner,
	title string,
	message string,
	detail string,
	confirmLabel string,
	confirmAction func(),
	danger bool,
) Dialog {
	controls := host.Components().Modal()
	dialog := &confirmDialog{confirmAction: confirmAction}
	messageView := controls.TextView()
	messageView.SetText(message)
	messageView.SetTextAlign(tview.AlignCenter)
	messageView.SetWrap(true)
	messageView.SetWordWrap(true)
	detailView := controls.TextView()
	detailView.SetText(detail)
	detailView.SetStyle(components.TextViewMuted)
	detailView.SetTextAlign(tview.AlignCenter)
	if danger {
		dialog.confirm = controls.DangerButton(confirmLabel)
	} else {
		dialog.confirm = controls.Button(confirmLabel)
	}
	dialog.cancel = controls.Button("Cancel")
	dialog.confirm.SetSelectedFunc(dialog.submit)
	dialog.cancel.SetSelectedFunc(dialog.close)
	buttons := controls.Flex(tview.FlexColumn).
		AddItem(nil, 0, 1, false).
		AddItem(dialog.cancel, 12, 0, false).
		AddItem(nil, 2, 0, false).
		AddItem(dialog.confirm, 12, 0, false).
		AddItem(nil, 0, 1, false)
	dialog.Layout = NewLayout(controls, title, 58).
		Row(messageView, 1).
		Row(detailView, 1).
		Spacer().
		Actions(buttons)
	dialog.handle = host.OpenModal(owner, dialog)
	return dialog
}

func (d *confirmDialog) Focusables() []tview.Primitive {
	return []tview.Primitive{d.cancel, d.confirm}
}

func (d *confirmDialog) KeyBindings() []keybinding.Binding { return nil }

func (d *confirmDialog) submit() {
	d.close()
	if d.confirmAction != nil {
		d.confirmAction()
	}
}

func (d *confirmDialog) close() {
	if d.handle != nil {
		d.handle.Close()
	}
}
