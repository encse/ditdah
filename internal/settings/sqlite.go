package settings

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/go-jet/jet/v2/qrm"
	. "github.com/go-jet/jet/v2/sqlite"
	dbmodel "morsemanual/internal/database/dbgen/model"
	. "morsemanual/internal/database/dbgen/table"
)

const settingsID int64 = 1

type sqliteStore struct {
	db *sql.DB
}

// NewSQLiteStore exposes settings persistence over the application-owned database.
func NewSQLiteStore(db *sql.DB) Store {
	return &sqliteStore{db: db}
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
		))

	if _, err := statement.ExecContext(ctx, s.db); err != nil {
		return Settings{}, fmt.Errorf("save application settings: %w", err)
	}

	return settings, nil
}

func settingsFromModel(stored dbmodel.ApplicationSettings) Settings {
	return Settings{
		StationCallsign:    stored.StationCallsign,
		QRZPassword:        stored.QrzPassword,
		QRZAPIKey:          stored.QrzAPIKey,
		MorseInputDeviceID: stored.MorseInputDeviceID,
	}
}

func settingsToModel(settings Settings) dbmodel.ApplicationSettings {
	return dbmodel.ApplicationSettings{
		ID:                 settingsID,
		StationCallsign:    settings.StationCallsign,
		QrzPassword:        settings.QRZPassword,
		QrzAPIKey:          settings.QRZAPIKey,
		MorseInputDeviceID: settings.MorseInputDeviceID,
	}
}
