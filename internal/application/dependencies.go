package application

import (
	"database/sql"

	"morsemanual/internal/qrz"
	"morsemanual/internal/stores"
)

type dependencies struct {
	stores stores.Stores
	qrz    qrz.Service
}

func newDependencies(db *sql.DB) dependencies {
	return dependencies{
		stores: stores.New(db),
		qrz:    qrz.New(),
	}
}
