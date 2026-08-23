//go:build screenshots

package tui

import (
	"testing"

	"ditdah/internal/callsign"
	"ditdah/internal/optional"
	"ditdah/internal/screenshots"
	ui "ditdah/internal/tui"
	"ditdah/internal/tui/keybinding"

	"github.com/gdamore/tcell/v2"
)

func TestScreenshotMorseDecoder(t *testing.T) {
	app := ui.NewApplication()
	app.AddKeyBinding(keybinding.OnKey(tcell.KeyF1, "Morse decoder", func() {}))
	app.AddKeyBinding(keybinding.OnKey(tcell.KeyF2, "Logbook", func() {}))
	page := newPage(app, nil, nil, nil, nil, nil)
	page.statusText = "Listening: USB Audio CODEC"
	page.callsigns = []string{"OH2BH", "DL1ABC", "G3XYZ", "I2RTF"}
	page.selectedCallsign = "OH2BH"
	page.renderCallsigns()
	_, _ = page.decodedText.WriteString(
		"CQ CQ CQ DE OH2BH OH2BH K\n\n" +
			"OH2BH DE HA7NCS HA7NCS K\n\n" +
			"HA7NCS DE OH2BH GM ES TNX FER CALL\n" +
			"UR RST 579 579 NAME MARTTI QTH KIRKKONUMMI\n" +
			"WX SUNNY TEMP 22C HW CPY? HA7NCS DE OH2BH K\n\n" +
			"OH2BH DE HA7NCS FB MARTTI TNX 579\n" +
			"NAME PETER QTH BUDAPEST\n",
	)
	page.renderDecodedText()
	page.details.SetText(formatCallsignDetails(callsign.Entry{
		Status: callsign.StatusReady,
		Record: optional.Some(callsign.Record{
			Callsign: "OH2BH",
			Name:     optional.Some("Martti"),
			QTH:      optional.Some("Kirkkonummi"),
			Country:  optional.Some("Finland"),
			Grid:     optional.Some("KP20"),
			CQZone:   optional.Some("15"),
			ITUZone:  optional.Some("18"),
		}),
	}))

	screen := tcell.NewSimulationScreen("UTF-8")
	if err := screen.Init(); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(screen.Fini)
	screen.SetSize(148, 42)
	if err := ui.DrawScreenshot(app, page, nil, screen); err != nil {
		t.Fatal(err)
	}
	if err := screenshots.WriteSVG(
		screen,
		"../../../docs/screenshots/morse-decoder.svg",
		"ditdah — morse decoder",
	); err != nil {
		t.Fatal(err)
	}
}
