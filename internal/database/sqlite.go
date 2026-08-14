package database

import (
	"context"
	"database/sql"
	"embed"
	"fmt"
	"io/fs"

	"github.com/pressly/goose/v3"
	_ "modernc.org/sqlite"
)

//go:generate go run -tags tools ./generate

//go:embed migrations/*.sql
var migrations embed.FS

// OpenSQLite opens the application database and applies all pending schema
// migrations.
func OpenSQLite(path string) (*sql.DB, error) {
	return openSQLite(path)
}

// OpenMemory opens an isolated in-memory application database.
func OpenMemory() (*sql.DB, error) {
	return openSQLite(":memory:")
}

func openSQLite(dataSourceName string) (*sql.DB, error) {
	db, err := sql.Open("sqlite", dataSourceName)
	if err != nil {
		return nil, fmt.Errorf("open application database: %w", err)
	}

	db.SetMaxOpenConns(1)
	db.SetMaxIdleConns(1)

	if err := initialize(context.Background(), db); err != nil {
		db.Close()
		return nil, err
	}

	return db, nil
}

func initialize(ctx context.Context, db *sql.DB) error {
	settings := []string{
		"PRAGMA foreign_keys = ON",
		"PRAGMA journal_mode = WAL",
		"PRAGMA busy_timeout = 5000",
	}
	for _, setting := range settings {
		if _, err := db.ExecContext(ctx, setting); err != nil {
			return fmt.Errorf("configure application database: %w", err)
		}
	}

	migrationFS, err := fs.Sub(migrations, "migrations")
	if err != nil {
		return fmt.Errorf("open database migrations: %w", err)
	}
	provider, err := goose.NewProvider(goose.DialectSQLite3, db, migrationFS)
	if err != nil {
		return fmt.Errorf("prepare database migrations: %w", err)
	}
	if _, err := provider.Up(ctx); err != nil {
		return fmt.Errorf("migrate application database: %w", err)
	}

	return nil
}
