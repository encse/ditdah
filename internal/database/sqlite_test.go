package database

import (
	"context"
	"database/sql"
	"path/filepath"
	"testing"
)

func TestOpenSQLiteAppliesMigrationsOnce(t *testing.T) {
	path := filepath.Join(t.TempDir(), "application.db")

	for attempt := 0; attempt < 2; attempt++ {
		db, err := OpenSQLite(path)
		if err != nil {
			t.Fatalf("OpenSQLite() attempt %d error = %v", attempt+1, err)
		}

		assertSchemaVersion(t, db, 1)
		assertTableExists(t, db, "qso")

		if err := db.Close(); err != nil {
			t.Fatalf("Close() attempt %d error = %v", attempt+1, err)
		}
	}
}

func assertSchemaVersion(t *testing.T, db *sql.DB, want int64) {
	t.Helper()

	rows, err := db.QueryContext(
		context.Background(),
		"SELECT MAX(version_id) FROM goose_db_version WHERE is_applied = 1",
	)
	if err != nil {
		t.Fatalf("query schema version: %v", err)
	}
	defer rows.Close()

	if !rows.Next() {
		t.Fatal("schema version query returned no rows")
	}
	var got int64
	if err := rows.Scan(&got); err != nil {
		t.Fatalf("scan schema version: %v", err)
	}
	if got != want {
		t.Fatalf("schema version = %d, want %d", got, want)
	}
}

func assertTableExists(t *testing.T, db *sql.DB, table string) {
	t.Helper()

	rows, err := db.QueryContext(
		context.Background(),
		"SELECT COUNT(*) FROM sqlite_master WHERE type = 'table' AND name = ?",
		table,
	)
	if err != nil {
		t.Fatalf("query table %q: %v", table, err)
	}
	defer rows.Close()

	if !rows.Next() {
		t.Fatalf("table query for %q returned no rows", table)
	}
	var count int
	if err := rows.Scan(&count); err != nil {
		t.Fatalf("scan table %q count: %v", table, err)
	}
	if count != 1 {
		t.Fatalf("table %q count = %d, want 1", table, count)
	}
}
