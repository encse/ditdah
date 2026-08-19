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

	terminal, err := newTerminalApplication(ctx, deps)
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
	logbook, err := logbookpage.New(
		ctx,
		terminal,
		deps.stores.Logbook,
		deps.qrzSync,
	)
	if err != nil {
		return nil, err
	}
	decoder := decoderpage.New(terminal)
	for _, page := range []tui.Page{logbook, decoder} {
		if err := terminal.Register(page); err != nil {
			return nil, err
		}
	}

	terminal.AddMenuItem(
		"Logbook",
		keybinding.OnKey(tcell.KeyF1, "logbook", func() {
			_ = terminal.Show(logbook.ID())
		}),
	)
	terminal.AddMenuItem(
		"Morse decoder",
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
	terminal.AddMenuItem(
		"Morse input",
		keybinding.OnRune('i', "Morse input", func() {
			settingspage.OpenMorseInput(
				ctx,
				terminal,
				deps.audio,
				deps.stores.Settings,
			)
		}),
	)
	if err := terminal.Show(logbook.ID()); err != nil {
		return nil, err
	}
	return terminal, nil
}
