//go:build tools

package main

import (
	"context"
	"database/sql"
	"fmt"
	"os"

	sqlitegen "github.com/go-jet/jet/v2/generator/sqlite"
	"github.com/pressly/goose/v3"
	_ "modernc.org/sqlite"
)

func main() {
	database, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		fail("open schema database", err)
	}
	defer database.Close()

	provider, err := goose.NewProvider(
		goose.DialectSQLite3,
		database,
		os.DirFS("migrations"),
	)
	if err != nil {
		fail("prepare schema migrations", err)
	}
	if _, err := provider.Up(context.Background()); err != nil {
		fail("apply schema migrations", err)
	}
	// Goose's version table is migration infrastructure, not part of the
	// application query model generated for persistence modules.
	if _, err := database.Exec("DROP TABLE goose_db_version"); err != nil {
		fail("remove migration metadata from generated schema", err)
	}

	if err := sqlitegen.GenerateDB(database, "dbgen"); err != nil {
		fail("generate Jet schema", err)
	}
}

func fail(operation string, err error) {
	fmt.Fprintf(os.Stderr, "%s: %v\n", operation, err)
	os.Exit(1)
}
