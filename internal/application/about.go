package application

import (
	ui "ditdah/internal/tui"
	"ditdah/internal/tui/components"
	"ditdah/internal/tui/keybinding"
	"ditdah/internal/tui/modal"

	"github.com/rivo/tview"
)

const website = "https://encse.github.io/ditdah/"

type aboutDialog struct {
	modal.Layout
	ok     components.Button
	handle modal.Handle
}

func openAbout(host ui.Application, version string) {
	controls := host.Components().Modal()
	dialog := &aboutDialog{}

	name := controls.TextView()
	name.SetText("DitDah")
	name.SetStyle(components.TextViewAccent)
	name.SetTextAlign(tview.AlignCenter)

	versionView := controls.TextView()
	versionView.SetText(version)
	versionView.SetStyle(components.TextViewMuted)
	versionView.SetTextAlign(tview.AlignCenter)

	developer := controls.TextView()
	developer.SetText("Developed by HA7NCS")
	developer.SetTextAlign(tview.AlignCenter)

	websiteView := controls.TextView()
	websiteView.SetDynamicColors(true)
	websiteView.SetText("[:::" + website + "]" + website + "[:::-]")
	websiteView.SetStyle(components.TextViewAccent)
	websiteView.SetTextAlign(tview.AlignCenter)

	dialog.ok = controls.Button("OK")
	dialog.ok.SetSelectedFunc(dialog.close)
	buttons := controls.Flex(tview.FlexColumn).
		AddItem(nil, 0, 1, false).
		AddItem(dialog.ok, 12, 0, false).
		AddItem(nil, 0, 1, false)

	dialog.Layout = modal.NewLayout(controls, " About ", 48).
		Row(name, 1).
		Row(versionView, 1).
		Spacer().
		Row(developer, 1).
		Row(websiteView, 1).
		Spacer().
		Actions(buttons)
	dialog.handle = host.OpenModalForCurrentLayer(dialog)
}

func (d *aboutDialog) Focusables() []tview.Primitive {
	return []tview.Primitive{d.ok}
}

func (d *aboutDialog) KeyBindings() []keybinding.Binding { return nil }

func (d *aboutDialog) close() {
	if d.handle != nil {
		d.handle.Close()
	}
}
