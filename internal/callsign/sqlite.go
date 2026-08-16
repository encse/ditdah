package callsign

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/go-jet/jet/v2/qrm"
	. "github.com/go-jet/jet/v2/sqlite"
	dbmodel "morsemanual/internal/database/dbgen/model"
	. "morsemanual/internal/database/dbgen/table"
	"morsemanual/internal/optional"
)

type sqliteStore struct {
	db *sql.DB
}

// NewSQLiteStore exposes callsign cache persistence over the shared database.
func NewSQLiteStore(db *sql.DB) Store {
	return &sqliteStore{db: db}
}

func (s *sqliteStore) Lookup(ctx context.Context, callsign string) (Entry, error) {
	statement := SELECT(CallsignLookup.AllColumns).
		FROM(CallsignLookup).
		WHERE(CallsignLookup.Callsign.EQ(String(normalizeCallsign(callsign)))).
		LIMIT(1)

	var stored dbmodel.CallsignLookup
	if err := statement.QueryContext(ctx, s.db, &stored); errors.Is(err, qrm.ErrNoRows) {
		return Entry{}, ErrNotFound
	} else if err != nil {
		return Entry{}, fmt.Errorf("lookup cached callsign: %w", err)
	}

	entry, err := entryFromModel(stored)
	if err != nil {
		return Entry{}, fmt.Errorf("decode cached callsign: %w", err)
	}
	return entry, nil
}

func (s *sqliteStore) Save(ctx context.Context, entry Entry) (Entry, error) {
	entry = normalize(entry)
	if err := validate(entry); err != nil {
		return Entry{}, err
	}

	stored, err := entryToModel(entry)
	if err != nil {
		return Entry{}, fmt.Errorf("encode cached callsign: %w", err)
	}
	statement := CallsignLookup.
		INSERT(CallsignLookup.AllColumns).
		MODEL(stored).
		ON_CONFLICT(CallsignLookup.Callsign).
		DO_UPDATE(SET(
			CallsignLookup.QueryCallsign.SET(CallsignLookup.EXCLUDED.QueryCallsign),
			CallsignLookup.Status.SET(CallsignLookup.EXCLUDED.Status),
			CallsignLookup.SavedAtUnixMs.SET(CallsignLookup.EXCLUDED.SavedAtUnixMs),
			CallsignLookup.Error.SET(CallsignLookup.EXCLUDED.Error),
			CallsignLookup.ProvidersChecked.SET(CallsignLookup.EXCLUDED.ProvidersChecked),
			CallsignLookup.RecordCallsign.SET(CallsignLookup.EXCLUDED.RecordCallsign),
			CallsignLookup.Country.SET(CallsignLookup.EXCLUDED.Country),
			CallsignLookup.CqZone.SET(CallsignLookup.EXCLUDED.CqZone),
			CallsignLookup.Grid.SET(CallsignLookup.EXCLUDED.Grid),
			CallsignLookup.ItuZone.SET(CallsignLookup.EXCLUDED.ItuZone),
			CallsignLookup.Name.SET(CallsignLookup.EXCLUDED.Name),
			CallsignLookup.Nickname.SET(CallsignLookup.EXCLUDED.Nickname),
			CallsignLookup.QrzURL.SET(CallsignLookup.EXCLUDED.QrzURL),
			CallsignLookup.Qth.SET(CallsignLookup.EXCLUDED.Qth),
			CallsignLookup.State.SET(CallsignLookup.EXCLUDED.State),
		))

	if _, err := statement.ExecContext(ctx, s.db); err != nil {
		return Entry{}, fmt.Errorf("save cached callsign: %w", err)
	}
	return entry, nil
}

func entryFromModel(stored dbmodel.CallsignLookup) (Entry, error) {
	var providers []string
	if err := json.Unmarshal([]byte(stored.ProvidersChecked), &providers); err != nil {
		return Entry{}, err
	}

	entry := Entry{
		Callsign:         stored.Callsign,
		QueryCallsign:    stored.QueryCallsign,
		Status:           stored.Status,
		SavedAt:          time.UnixMilli(stored.SavedAtUnixMs).UTC(),
		Error:            stored.Error,
		ProvidersChecked: providers,
	}
	if stored.RecordCallsign != nil {
		entry.Record = optional.Some(Record{
			Callsign: *stored.RecordCallsign,
			Country:  optionalString(stored.Country),
			CQZone:   optionalString(stored.CqZone),
			Grid:     optionalString(stored.Grid),
			ITUZone:  optionalString(stored.ItuZone),
			Name:     optionalString(stored.Name),
			Nickname: optionalString(stored.Nickname),
			QRZURL:   optionalString(stored.QrzURL),
			QTH:      optionalString(stored.Qth),
			State:    optionalString(stored.State),
		})
	}
	return entry, nil
}

func entryToModel(entry Entry) (dbmodel.CallsignLookup, error) {
	providers, err := json.Marshal(entry.ProvidersChecked)
	if err != nil {
		return dbmodel.CallsignLookup{}, err
	}
	stored := dbmodel.CallsignLookup{
		Callsign:         entry.Callsign,
		QueryCallsign:    entry.QueryCallsign,
		Status:           entry.Status,
		SavedAtUnixMs:    entry.SavedAt.UnixMilli(),
		Error:            entry.Error,
		ProvidersChecked: string(providers),
	}
	if record, present := entry.Record.Get(); present {
		stored.RecordCallsign = &record.Callsign
		stored.Country = optionalPointer(record.Country)
		stored.CqZone = optionalPointer(record.CQZone)
		stored.Grid = optionalPointer(record.Grid)
		stored.ItuZone = optionalPointer(record.ITUZone)
		stored.Name = optionalPointer(record.Name)
		stored.Nickname = optionalPointer(record.Nickname)
		stored.QrzURL = optionalPointer(record.QRZURL)
		stored.Qth = optionalPointer(record.QTH)
		stored.State = optionalPointer(record.State)
	}
	return stored, nil
}

func optionalString(value *string) optional.Value[string] {
	if value == nil {
		return optional.None[string]()
	}
	return optional.Some(*value)
}

func optionalPointer(value optional.Value[string]) *string {
	stored, present := value.Get()
	if !present {
		return nil
	}
	return &stored
}
