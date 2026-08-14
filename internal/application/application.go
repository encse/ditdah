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
	if err := tui.Run(ctx, store); err != nil {
		return fmt.Errorf("run terminal UI: %w", err)
	}

	return nil
}
