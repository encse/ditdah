package application

import (
	"database/sql"

	"morsemanual/internal/audio"
	"morsemanual/internal/callsign"
	"morsemanual/internal/qrz"
	"morsemanual/internal/stores"
)

type dependencies struct {
	stores         stores.Stores
	qrz            qrz.Service
	qrzSync        qrz.Synchronizer
	callsignLookup callsign.Service
	audio          audio.Source
}

func newDependencies(db *sql.DB) (dependencies, error) {
	allStores := stores.New(db)
	client := qrz.New()
	input, err := audio.NewInput(audio.DefaultConfig())
	if err != nil {
		return dependencies{}, err
	}
	return dependencies{
		stores: allStores,
		qrz:    client,
		qrzSync: qrz.NewSynchronizer(
			client,
			allStores.Logbook,
			allStores.Settings,
		),
		callsignLookup: callsign.NewService(
			allStores.Callsign,
			allStores.Settings,
			client,
		),
		audio: input,
	}, nil
}

func (d dependencies) close() error {
	if d.audio == nil {
		return nil
	}
	return d.audio.Close()
}
