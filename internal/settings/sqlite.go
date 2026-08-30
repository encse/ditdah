package settings

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"ditdah/internal/syncutil"

	dbmodel "ditdah/internal/database/dbgen/model"
	. "ditdah/internal/database/dbgen/table"
	"github.com/go-jet/jet/v2/qrm"
	. "github.com/go-jet/jet/v2/sqlite"
)

const settingsID int64 = 1

type sqliteStore struct {
	db      *sql.DB
	changes syncutil.Broadcaster
}

// NewSQLiteStore exposes settings persistence over the application-owned database.
func NewSQLiteStore(db *sql.DB) Store {
	return &sqliteStore{
		db:      db,
		changes: syncutil.NewBroadcaster(),
	}
}

func (s *sqliteStore) Subscribe() syncutil.Subscription {
	return s.changes.Subscribe()
}

func (s *sqliteStore) Load(ctx context.Context) (Settings, error) {
	statement := SELECT(ApplicationSettings.AllColumns).
		FROM(ApplicationSettings).
		WHERE(ApplicationSettings.ID.EQ(Int(settingsID))).
		LIMIT(1)

	var stored dbmodel.ApplicationSettings
	if err := statement.QueryContext(ctx, s.db, &stored); errors.Is(err, qrm.ErrNoRows) {
		return Settings{}, nil
	} else if err != nil {
		return Settings{}, fmt.Errorf("load application settings: %w", err)
	}

	return settingsFromModel(stored), nil
}

func (s *sqliteStore) Save(
	ctx context.Context,
	settings Settings,
) (Settings, error) {
	settings = normalize(settings)
	settings.Configured = true
	stored := settingsToModel(settings)
	statement := ApplicationSettings.
		INSERT(ApplicationSettings.AllColumns).
		MODEL(stored).
		ON_CONFLICT(ApplicationSettings.ID).
		DO_UPDATE(SET(
			ApplicationSettings.StationCallsign.SET(
				ApplicationSettings.EXCLUDED.StationCallsign,
			),
			ApplicationSettings.QrzPassword.SET(
				ApplicationSettings.EXCLUDED.QrzPassword,
			),
			ApplicationSettings.QrzAPIKey.SET(
				ApplicationSettings.EXCLUDED.QrzAPIKey,
			),
			ApplicationSettings.MorseInputDeviceID.SET(
				ApplicationSettings.EXCLUDED.MorseInputDeviceID,
			),
			ApplicationSettings.RadioModelID.SET(
				ApplicationSettings.EXCLUDED.RadioModelID,
			),
			ApplicationSettings.RadioModelName.SET(
				ApplicationSettings.EXCLUDED.RadioModelName,
			),
			ApplicationSettings.RadioSerialPort.SET(
				ApplicationSettings.EXCLUDED.RadioSerialPort,
			),
			ApplicationSettings.RadioBaudRate.SET(
				ApplicationSettings.EXCLUDED.RadioBaudRate,
			),
			ApplicationSettings.Configured.SET(
				ApplicationSettings.EXCLUDED.Configured,
			),
		))

	if _, err := statement.ExecContext(ctx, s.db); err != nil {
		return Settings{}, fmt.Errorf("save application settings: %w", err)
	}

	s.changes.Activate()
	return settings, nil
}

func settingsFromModel(stored dbmodel.ApplicationSettings) Settings {
	return Settings{
		StationCallsign:    stored.StationCallsign,
		QRZPassword:        stored.QrzPassword,
		QRZAPIKey:          stored.QrzAPIKey,
		MorseInputDeviceID: stored.MorseInputDeviceID,
		RadioModelID:       int(stored.RadioModelID),
		RadioModelName:     stored.RadioModelName,
		RadioSerialPort:    stored.RadioSerialPort,
		RadioBaudRate:      int(stored.RadioBaudRate),
		Configured:         stored.Configured != 0,
	}
}

func settingsToModel(settings Settings) dbmodel.ApplicationSettings {
	configured := int64(0)
	if settings.Configured {
		configured = 1
	}
	return dbmodel.ApplicationSettings{
		ID:                 settingsID,
		StationCallsign:    settings.StationCallsign,
		QrzPassword:        settings.QRZPassword,
		QrzAPIKey:          settings.QRZAPIKey,
		MorseInputDeviceID: settings.MorseInputDeviceID,
		RadioModelID:       int64(settings.RadioModelID),
		RadioModelName:     settings.RadioModelName,
		RadioSerialPort:    settings.RadioSerialPort,
		RadioBaudRate:      int64(settings.RadioBaudRate),
		Configured:         configured,
	}
}
