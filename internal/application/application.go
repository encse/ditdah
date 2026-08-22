// Package application wires the application's database, domain stores, and
// terminal user interface together.
package application

import (
	"context"
	"errors"
	"fmt"

	"morsemanual/internal/database"
	decoderpage "morsemanual/internal/decoder/tui"
	logbookdomain "morsemanual/internal/logbook"
	logbookpage "morsemanual/internal/logbook/tui"
	qsoeditor "morsemanual/internal/qsoeditor/tui"
	settingspage "morsemanual/internal/settings/tui"
	"morsemanual/internal/tui"
	"morsemanual/internal/tui/keybinding"

	"github.com/gdamore/tcell/v2"
)

// Run opens the application database and runs the terminal UI until the user
// quits or ctx is cancelled.
func Run(ctx context.Context, databasePath string) (err error) {
	db, err := database.OpenSQLite(databasePath)
	if err != nil {
		return err
	}
	defer func() {
		err = errors.Join(err, db.Close())
	}()

	deps, err := newDependencies(db)
	if err != nil {
		return fmt.Errorf("initialize dependencies: %w", err)
	}
	defer func() {
		err = errors.Join(err, deps.close())
	}()

	app, initialPageID, err := newTerminalApplication(deps)
	if err != nil {
		return err
	}
	if err := app.Run(ctx, initialPageID); err != nil {
		return fmt.Errorf("run terminal UI: %w", err)
	}

	return nil
}

func newTerminalApplication(
	deps dependencies,
) (tui.Application, string, error) {
	app := tui.NewApplication()
	var logbook logbookpage.Page
	qsoEditors := qsoeditor.New(
		app,
		deps.stores.Logbook,
		deps.stores.Settings,
		deps.callsignLookup,
		func(qso logbookdomain.QSO) {
			if logbook != nil {
				logbook.QSOChanged(qso)
			}
		},
	)
	logbook = logbookpage.New(
		app,
		deps.stores.Logbook,
		deps.qrzSync,
		qsoEditors.Create,
		qsoEditors.Edit,
	)
	decoder := decoderpage.New(
		app,
		deps.audio,
		deps.stores.Settings,
		deps.callsignLookup,
		qsoEditors.Create,
	)
	for _, page := range []tui.Page{logbook, decoder} {
		if err := app.Register(page); err != nil {
			return nil, "", err
		}
	}

	app.AddKeyBinding(
		keybinding.OnKey(tcell.KeyF1, "Logbook", func() {
			_ = app.Show(logbook.ID())
		}),
	)
	app.AddKeyBinding(
		keybinding.OnKey(tcell.KeyF2, "Morse decoder", func() {
			_ = app.Show(decoder.ID())
		}),
	)
	app.AddMenuItem(
		"Settings",
		keybinding.OnRune('s', "settings", func() {
			settingspage.Open(
				app,
				deps.stores.Settings,
				deps.qrz,
				deps.audio,
			)
		}),
	)
	return app, logbook.ID(), nil
}
