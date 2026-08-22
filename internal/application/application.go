// Package application wires the application's database, domain stores, and
// terminal user interface together.
package application

import (
	"context"
	"errors"
	"fmt"

	"morsemanual/internal/database"
	decoderpage "morsemanual/internal/decoder/tui"
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

	app, initialPage := newTerminalApplication(deps)
	if err := app.Run(ctx, initialPage); err != nil {
		return fmt.Errorf("run terminal UI: %w", err)
	}

	return nil
}

func newTerminalApplication(
	deps dependencies,
) (tui.Application, tui.Page) {
	app := tui.NewApplication()
	qsoEditors := qsoeditor.New(
		app,
		deps.stores.Logbook,
		deps.stores.Settings,
		deps.callsignLookup,
	)
	newLogbookPage := func() tui.Page {
		return logbookpage.New(
			app,
			deps.stores.Logbook,
			deps.qrzSync,
			qsoEditors.Create,
			qsoEditors.Edit,
		)
	}
	newDecoderPage := func() tui.Page {
		return decoderpage.New(
			app,
			deps.audio,
			deps.stores.Settings,
			deps.callsignLookup,
			qsoEditors.Create,
		)
	}

	app.AddKeyBinding(
		keybinding.OnKey(tcell.KeyF1, "Logbook", func() {
			_ = app.Show(newLogbookPage())
		}),
	)
	app.AddKeyBinding(
		keybinding.OnKey(tcell.KeyF2, "Morse decoder", func() {
			_ = app.Show(newDecoderPage())
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
	return app, newLogbookPage()
}
