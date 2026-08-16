// Package application wires the application's database, domain stores, and
// terminal user interface together.
package application

import (
	"context"
	"errors"
	"fmt"

	"morsemanual/internal/database"
	logbookpage "morsemanual/internal/logbook/tui"
	settingspage "morsemanual/internal/settings/tui"
	"morsemanual/internal/tui"
	"morsemanual/internal/tui/keybinding"
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

	terminal, err := newTerminalApplication(ctx, newDependencies(db))
	if err != nil {
		return err
	}
	if err := terminal.Run(ctx); err != nil {
		return fmt.Errorf("run terminal UI: %w", err)
	}

	return nil
}

func newTerminalApplication(
	ctx context.Context,
	deps dependencies,
) (tui.Application, error) {
	terminal := tui.NewApplication()
	terminal.AddMenuItem(
		"Settings",
		keybinding.OnRune('s', "settings", func() {
			settingspage.Open(
				ctx,
				terminal,
				deps.stores.Settings,
				deps.qrz,
			)
		}),
	)
	page, err := logbookpage.New(ctx, terminal, deps.stores.Logbook)
	if err != nil {
		return nil, err
	}
	if err := terminal.Register(page); err != nil {
		return nil, err
	}
	if err := terminal.Show(page.ID()); err != nil {
		return nil, err
	}
	return terminal, nil
}
