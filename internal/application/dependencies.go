package application

import (
	"database/sql"

	"ditdah/internal/audio"
	"ditdah/internal/callsign"
	"ditdah/internal/qrz"
	"ditdah/internal/radio"
	"ditdah/internal/stores"
)

type dependencies struct {
	stores         stores.Stores
	qrz            qrz.Service
	qrzSync        qrz.Synchronizer
	callsignLookup callsign.Service
	audio          audio.Source
	radio          radio.Monitor
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
		radio: radio.NewMonitor(allStores.Settings, radio.New()),
	}, nil
}

func (d dependencies) close() error {
	if d.audio == nil {
		return nil
	}
	return d.audio.Close()
}
