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
	"morsemanual/internal/stores"
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

	applicationStores := stores.New(db)
	terminal, err := newTerminalApplication(ctx, applicationStores)
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
	applicationStores stores.Stores,
) (tui.Application, error) {
	terminal := tui.NewApplication()
	terminal.AddMenuItem(
		"Settings",
		keybinding.OnRune('s', "settings", func() {
			settingspage.Open(ctx, terminal, applicationStores.Settings)
		}),
	)
	page, err := logbookpage.New(ctx, terminal, applicationStores.Logbook)
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
