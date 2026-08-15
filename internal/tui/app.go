// Package tui implements the terminal application.
package tui

import (
	"context"

	domain "morsemanual/internal/logbook"
	logbookpage "morsemanual/internal/tui/logbook"
)

// Run assembles and runs the terminal application.
func Run(ctx context.Context, store domain.Store) error {
	app, err := newApp(ctx, store)
	if err != nil {
		return err
	}
	return app.Run(ctx)
}

func newApp(ctx context.Context, store domain.Store) (Application, error) {
	app := NewApplication(nordTheme)
	logbook, err := logbookpage.New(ctx, app, store)
	if err != nil {
		return nil, err
	}
	if err := app.Register(logbook); err != nil {
		return nil, err
	}
	if err := app.Show(logbook.ID()); err != nil {
		return nil, err
	}
	return app, nil
}
