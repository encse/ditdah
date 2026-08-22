//go:build screenshots

package tui

import (
	"testing"
	"time"

	domain "ditdah/internal/logbook"
	"ditdah/internal/optional"
	"ditdah/internal/screenshots"
	ui "ditdah/internal/tui"
	"ditdah/internal/tui/keybinding"

	"github.com/gdamore/tcell/v2"
)

func TestScreenshotLogbook(t *testing.T) {
	app := ui.NewApplication()
	addScreenshotNavigation(app)
	page := newPage(app, nil, nil, func(ui.Owner, string) {}, func(ui.Owner, domain.QSO) {})
	page.qsos = screenshotQSOs()
	page.applyFilter()

	screen := screenshotScreen(t)
	if err := ui.DrawScreenshot(app, page, nil, screen); err != nil {
		t.Fatal(err)
	}
	if err := screenshots.WriteSVG(
		screen,
		"../../../docs/screenshots/logbook.svg",
		"ditdah — logbook",
	); err != nil {
		t.Fatal(err)
	}
}

func screenshotScreen(t *testing.T) tcell.SimulationScreen {
	t.Helper()
	screen := tcell.NewSimulationScreen("UTF-8")
	if err := screen.Init(); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(screen.Fini)
	screen.SetSize(148, 42)
	return screen
}

func addScreenshotNavigation(app ui.Application) {
	app.AddKeyBinding(keybinding.OnKey(tcell.KeyF1, "Logbook", func() {}))
	app.AddKeyBinding(keybinding.OnKey(tcell.KeyF2, "Morse decoder", func() {}))
}

func screenshotQSOs() []domain.QSO {
	zone := time.FixedZone("CEST", 2*60*60)
	contact := func(day, hour, minute int) time.Time {
		return time.Date(2026, time.August, day, hour, minute, 0, 0, zone)
	}
	synced := func(value time.Time) optional.Value[time.Time] {
		return optional.Some(value.Add(2 * time.Minute))
	}
	return []domain.QSO{
		{
			ID: "q1", StationCallsign: "HA7NCS", Callsign: "OH2BH",
			StartedAt: contact(21, 18, 40), FrequencyHz: optional.Some[int64](14_025_000),
			Mode: "CW", RSTSent: "599", RSTReceived: "599", Name: "Martti",
			QTH: "Kirkkonummi, Finland", Notes: "Relaxed evening QSO on 20 metres.",
			QRZSyncedAt: synced(contact(21, 18, 40)),
		},
		{
			ID: "q2", StationCallsign: "HA7NCS", Callsign: "DL1ABC",
			StartedAt: contact(21, 17, 5), FrequencyHz: optional.Some[int64](7_023_000),
			Mode: "CW", RSTSent: "579", RSTReceived: "589", Name: "Anna",
			QTH: "Berlin, Germany", Notes: "Some QSB, easy copy after narrowing the filter.",
		},
		{
			ID: "q3", StationCallsign: "HA7NCS", Callsign: "G3XYZ",
			StartedAt: contact(20, 16, 45), FrequencyHz: optional.Some[int64](21_030_000),
			Mode: "CW", RSTSent: "589", RSTReceived: "599", Name: "James",
			QTH: "Bristol, England", QRZSyncedAt: synced(contact(20, 16, 45)),
		},
		{
			ID: "q4", StationCallsign: "HA7NCS", Callsign: "I2RTF",
			StartedAt: contact(19, 15, 20), FrequencyHz: optional.Some[int64](10_118_000),
			Mode: "CW", RSTSent: "559", RSTReceived: "579", Name: "Luca", QTH: "Milan, Italy",
		},
		{
			ID: "q5", StationCallsign: "HA7NCS", Callsign: "F5JBR",
			StartedAt: contact(18, 14, 5), FrequencyHz: optional.Some[int64](18_082_000),
			Mode: "CW", RSTSent: "589", RSTReceived: "599", Name: "Jean", QTH: "Lyon, France",
			QRZSyncedAt: synced(contact(18, 14, 5)),
		},
		{
			ID: "q6", StationCallsign: "HA7NCS", Callsign: "K1ZZ",
			StartedAt: contact(17, 13, 10), FrequencyHz: optional.Some[int64](14_042_000),
			Mode: "CW", RSTSent: "579", RSTReceived: "559", Name: "Dave", QTH: "Connecticut, USA",
		},
		{
			ID: "q7", StationCallsign: "HA7NCS", Callsign: "OE3KAB",
			StartedAt: contact(16, 20, 32), FrequencyHz: optional.Some[int64](3_558_000),
			Mode: "CW", RSTSent: "599", RSTReceived: "589", Name: "Karl", QTH: "Vienna, Austria",
			QRZSyncedAt: synced(contact(16, 20, 32)),
		},
		{
			ID: "q8", StationCallsign: "HA7NCS", Callsign: "SM5IMO",
			StartedAt: contact(15, 19, 18), FrequencyHz: optional.Some[int64](7_028_000),
			Mode: "CW", RSTSent: "579", RSTReceived: "579", Name: "Erik", QTH: "Uppsala, Sweden",
		},
	}
}
