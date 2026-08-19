package application

import (
	"database/sql"

	"morsemanual/internal/qrz"
	"morsemanual/internal/stores"
)

type dependencies struct {
	stores  stores.Stores
	qrz     qrz.Service
	qrzSync qrz.Synchronizer
}

func newDependencies(db *sql.DB) dependencies {
	allStores := stores.New(db)
	client := qrz.New()
	return dependencies{
		stores: allStores,
		qrz:    client,
		qrzSync: qrz.NewSynchronizer(
			client,
			allStores.Logbook,
			allStores.Settings,
		),
	}
}
