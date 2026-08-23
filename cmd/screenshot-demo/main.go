//go:build screenshots

package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	decoderpage "ditdah/internal/decoder/tui"
	logbookpage "ditdah/internal/logbook/tui"
	qsoeditor "ditdah/internal/qsoeditor/tui"
	"ditdah/internal/tui"
	"ditdah/internal/tui/components"
	"ditdah/internal/tui/keybinding"
	"ditdah/internal/tui/modal"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

func main() {
	if len(os.Args) != 2 {
		fail("usage: screenshot-demo decoder|logbook|new-qso")
	}

	app := tui.NewApplication()
	addNavigation(app)
	page, onStarted, err := screen(app, os.Args[1])
	if err != nil {
		fail(err.Error())
	}

	ctx, cancel := signal.NotifyContext(
		context.Background(),
		os.Interrupt,
		syscall.SIGTERM,
	)
	defer cancel()
	if err := app.Run(ctx, page, onStarted); err != nil {
		fail(err.Error())
	}
}

func screen(app tui.Application, name string) (tui.Page, func(), error) {
	switch name {
	case "decoder":
		return keepPageOpen(decoderpage.NewScreenshotPage(app)), nil, nil
	case "logbook":
		return keepPageOpen(logbookpage.NewScreenshotPage(app)), nil, nil
	case "new-qso":
		page := &blankPage{content: app.Components().TextView()}
		dialog := keepDialogOpen(qsoeditor.NewScreenshotDialog(app))
		return page, func() { app.OpenModalForCurrentLayer(dialog) }, nil
	default:
		return nil, nil, fmt.Errorf("unknown screenshot %q", name)
	}
}

func addNavigation(app tui.Application) {
	app.AddKeyBinding(keybinding.OnKey(
		tcell.KeyF1,
		"Morse decoder",
		func() {},
	))
	app.AddKeyBinding(keybinding.OnKey(
		tcell.KeyF2,
		"Logbook",
		func() {},
	))
}

type persistentPage struct {
	tui.Page
}

func keepPageOpen(page tui.Page) tui.Page {
	return &persistentPage{Page: page}
}

func (p *persistentPage) Run(ctx context.Context) { <-ctx.Done() }

type persistentDialog struct {
	modal.Dialog
}

func keepDialogOpen(dialog modal.Dialog) modal.Dialog {
	return &persistentDialog{Dialog: dialog}
}

func (d *persistentDialog) Run(ctx context.Context) { <-ctx.Done() }

type blankPage struct {
	content tview.Primitive
}

func (p *blankPage) ID() string                        { return "logbook" }
func (p *blankPage) Title() string                     { return "Logbook" }
func (p *blankPage) Content() tview.Primitive          { return p.content }
func (p *blankPage) Focusables() []tview.Primitive     { return nil }
func (p *blankPage) KeyBindings() []keybinding.Binding { return nil }
func (p *blankPage) MenuItems() []components.MenuItem  { return nil }
func (p *blankPage) Run(ctx context.Context)           { <-ctx.Done() }

func fail(message string) {
	fmt.Fprintln(os.Stderr, message)
	os.Exit(1)
}
