// Package stores constructs and groups the application's domain stores.
package stores

import (
	"database/sql"

	"ditdah/internal/callsign"
	"ditdah/internal/logbook"
	"ditdah/internal/settings"
)

// Stores contains the persistence boundaries shared by application features.
// It is assembled once from the application-owned database connection.
type Stores struct {
	Callsign callsign.Store
	Logbook  logbook.Store
	Settings settings.Store
}

// New creates all domain stores over the shared application database.
func New(db *sql.DB) Stores {
	return Stores{
		Callsign: callsign.NewSQLiteStore(db),
		Logbook:  logbook.NewSQLiteStore(db),
		Settings: settings.NewSQLiteStore(db),
	}
}
