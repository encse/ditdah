// Package application wires the application's database, domain stores, and
// terminal user interface together.
package application

import (
	"context"
	"errors"
	"fmt"

	"morsemanual/internal/database"
	"morsemanual/internal/logbook"
	"morsemanual/internal/tui"
	logbookpage "morsemanual/internal/tui/logbook"
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

	store := logbook.NewSQLiteStore(db)
	terminal, err := newTerminalApplication(ctx, store)
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
	store logbook.Store,
) (tui.Application, error) {
	terminal := tui.NewApplication()
	page, err := logbookpage.New(ctx, terminal, store)
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
