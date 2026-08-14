//go:build tools

package main

import (
	"database/sql"
	"fmt"
	"os"

	sqlitegen "github.com/go-jet/jet/v2/generator/sqlite"
	_ "modernc.org/sqlite"
)

func main() {
	database, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		fail("open schema database", err)
	}
	defer database.Close()

	schema, err := os.ReadFile("schema.sql")
	if err != nil {
		fail("read schema", err)
	}
	if _, err := database.Exec(string(schema)); err != nil {
		fail("apply schema", err)
	}

	if err := sqlitegen.GenerateDB(database, "dbgen"); err != nil {
		fail("generate Jet schema", err)
	}
}

func fail(operation string, err error) {
	fmt.Fprintf(os.Stderr, "%s: %v\n", operation, err)
	os.Exit(1)
}
