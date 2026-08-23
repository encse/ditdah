//go:build screenshots

package tui

import (
	"context"
	"testing"
	"time"

	domain "ditdah/internal/logbook"
	"ditdah/internal/optional"
	"ditdah/internal/screenshots"
	ui "ditdah/internal/tui"
	"ditdah/internal/tui/components"
	"ditdah/internal/tui/keybinding"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

func TestScreenshotNewQSO(t *testing.T) {
	app := ui.NewApplication()
	app.AddKeyBinding(keybinding.OnKey(tcell.KeyF1, "Morse decoder", func() {}))
	app.AddKeyBinding(keybinding.OnKey(tcell.KeyF2, "Logbook", func() {}))
	page := &screenshotPage{content: app.Components().TextView()}
	editor := newQSOEditor(app, screenshotQSO(), nil, nil)

	screen := tcell.NewSimulationScreen("UTF-8")
	if err := screen.Init(); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(screen.Fini)
	screen.SetSize(120, 42)
	if err := ui.DrawScreenshot(app, page, editor, screen); err != nil {
		t.Fatal(err)
	}
	if err := screenshots.WriteSVG(
		screen,
		"../../../docs/screenshots/new-qso.svg",
		"ditdah — new QSO",
	); err != nil {
		t.Fatal(err)
	}
}

func screenshotQSO() domain.QSO {
	return domain.QSO{
		StationCallsign: "HA7NCS",
		Callsign:        "OH2BH",
		StartedAt: time.Date(
			2026,
			time.August,
			21,
			18,
			40,
			0,
			0,
			time.FixedZone("CEST", 2*60*60),
		),
		FrequencyHz: optional.Some[int64](14_025_000),
		Mode:        "CW",
		RSTSent:     "599",
		RSTReceived: "599",
		Name:        "Martti",
		QTH:         "Kirkkonummi, Finland",
		Notes:       "Relaxed evening QSO on 20 metres.",
	}
}

type screenshotPage struct {
	content tview.Primitive
}

func (p *screenshotPage) ID() string                        { return "logbook" }
func (p *screenshotPage) Title() string                     { return "Logbook" }
func (p *screenshotPage) Content() tview.Primitive          { return p.content }
func (p *screenshotPage) Focusables() []tview.Primitive     { return nil }
func (p *screenshotPage) KeyBindings() []keybinding.Binding { return nil }
func (p *screenshotPage) MenuItems() []components.MenuItem  { return nil }
func (p *screenshotPage) Run(context.Context)               {}
