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

	terminal, initialPageID, err := newTerminalApplication(ctx, deps)
	if err != nil {
		return err
	}
	if err := terminal.Run(ctx, initialPageID); err != nil {
		return fmt.Errorf("run terminal UI: %w", err)
	}

	return nil
}

func newTerminalApplication(
	ctx context.Context,
	deps dependencies,
) (tui.Application, string, error) {
	terminal := tui.NewApplication()
	logbook, err := logbookpage.New(
		ctx,
		terminal,
		deps.stores.Logbook,
		deps.qrzSync,
	)
	if err != nil {
		return nil, "", err
	}
	decoder := decoderpage.New(
		terminal,
		deps.audio,
		deps.stores.Settings,
		deps.callsignLookup,
	)
	for _, page := range []tui.Page{logbook, decoder} {
		if err := terminal.Register(page); err != nil {
			return nil, "", err
		}
	}

	terminal.AddKeyBinding(
		keybinding.OnKey(tcell.KeyF1, "Logbook", func() {
			_ = terminal.Show(logbook.ID())
		}),
	)
	terminal.AddKeyBinding(
		keybinding.OnKey(tcell.KeyF2, "Morse decoder", func() {
			_ = terminal.Show(decoder.ID())
		}),
	)
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
	return terminal, logbook.ID(), nil
}
