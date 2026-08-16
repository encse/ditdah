// Package stores constructs and groups the application's domain stores.
package stores

import (
	"database/sql"

	"morsemanual/internal/logbook"
	"morsemanual/internal/settings"
)

// Stores contains the persistence boundaries shared by application features.
// It is assembled once from the application-owned database connection.
type Stores struct {
	Logbook  logbook.Store
	Settings settings.Store
}

// New creates all domain stores over the shared application database.
func New(db *sql.DB) Stores {
	return Stores{
		Logbook:  logbook.NewSQLiteStore(db),
		Settings: settings.NewSQLiteStore(db),
	}
}
