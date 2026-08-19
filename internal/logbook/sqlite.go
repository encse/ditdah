package logbook

import (
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/go-jet/jet/v2/qrm"
	. "github.com/go-jet/jet/v2/sqlite"
	dbmodel "morsemanual/internal/database/dbgen/model"
	. "morsemanual/internal/database/dbgen/table"
	"morsemanual/internal/optional"
)

const (
	defaultListLimit = 100
	maximumListLimit = 1000
)

type sqliteStore struct {
	db *sql.DB
}

// NewSQLiteStore exposes QSO persistence over the application-owned database.
func NewSQLiteStore(db *sql.DB) Store {
	return &sqliteStore{db: db}
}

func (s *sqliteStore) Add(ctx context.Context, qso QSO) (QSO, error) {
	qso = normalizeQSO(qso)
	if err := validateQSO(qso); err != nil {
		return QSO{}, err
	}

	if qso.ID == "" {
		id, err := newID()
		if err != nil {
			return QSO{}, err
		}
		qso.ID = id
	}

	now := currentTime()
	qso.CreatedAt = now
	qso.UpdatedAt = now

	statement := Qso.INSERT(Qso.AllColumns).
		MODEL(qsoToModel(qso))

	_, err := statement.ExecContext(ctx, s.db)
	if err != nil {
		return QSO{}, fmt.Errorf("add QSO: %w", err)
	}

	return qso, nil
}

func (s *sqliteStore) Get(ctx context.Context, id string) (QSO, error) {
	statement := SELECT(Qso.AllColumns).
		FROM(Qso).
		WHERE(Qso.ID.EQ(String(strings.TrimSpace(id)))).
		LIMIT(1)

	var stored dbmodel.Qso
	if err := statement.QueryContext(ctx, s.db, &stored); errors.Is(err, qrm.ErrNoRows) {
		return QSO{}, ErrNotFound
	} else if err != nil {
		return QSO{}, fmt.Errorf("get QSO: %w", err)
	}

	return qsoFromModel(stored), nil
}

func (s *sqliteStore) List(ctx context.Context, filter Filter) ([]QSO, error) {
	limit := filter.Limit
	if limit <= 0 {
		limit = defaultListLimit
	}
	if limit > maximumListLimit {
		limit = maximumListLimit
	}
	if filter.Offset < 0 {
		return nil, errors.New("offset cannot be negative")
	}

	statement := SELECT(Qso.AllColumns).
		FROM(Qso)

	if search := strings.TrimSpace(filter.Search); search != "" {
		pattern := String("%" + strings.ToLower(search) + "%")
		statement = statement.WHERE(OR(
			LOWER(Qso.StationCallsign).LIKE(pattern),
			LOWER(Qso.Callsign).LIKE(pattern),
			LOWER(Qso.Name).LIKE(pattern),
			LOWER(Qso.Qth).LIKE(pattern),
			LOWER(Qso.Mode).LIKE(pattern),
			LOWER(Qso.Notes).LIKE(pattern),
		))
	}

	statement = statement.
		ORDER_BY(Qso.StartedAtUnixMs.DESC(), Qso.ID.DESC()).
		LIMIT(int64(limit)).
		OFFSET(int64(filter.Offset))

	var stored []dbmodel.Qso
	if err := statement.QueryContext(ctx, s.db, &stored); err != nil {
		return nil, fmt.Errorf("list QSOs: %w", err)
	}

	qsos := make([]QSO, len(stored))
	for index, row := range stored {
		qsos[index] = qsoFromModel(row)
	}

	return qsos, nil
}

func (s *sqliteStore) Update(ctx context.Context, qso QSO) (QSO, error) {
	qso = normalizeQSO(qso)
	if qso.ID == "" {
		return QSO{}, errors.New("QSO id is required")
	}
	if err := validateQSO(qso); err != nil {
		return QSO{}, err
	}

	existing, err := s.Get(ctx, qso.ID)
	if err != nil {
		return QSO{}, err
	}

	qso.CreatedAt = existing.CreatedAt
	qso.UpdatedAt = currentTime()
	qso.QRZSyncedAt = optional.None[time.Time]()

	statement := Qso.
		UPDATE(Qso.MutableColumns.Except(Qso.CreatedAtUnixMs)).
		MODEL(qsoToModel(qso)).
		WHERE(Qso.ID.EQ(String(qso.ID)))

	result, err := statement.ExecContext(ctx, s.db)
	if err != nil {
		return QSO{}, fmt.Errorf("update QSO: %w", err)
	}

	changed, err := result.RowsAffected()
	if err != nil {
		return QSO{}, fmt.Errorf("read updated QSO count: %w", err)
	}
	if changed == 0 {
		return QSO{}, ErrNotFound
	}

	return qso, nil
}

func (s *sqliteStore) Delete(ctx context.Context, id string) error {
	statement := Qso.DELETE().
		WHERE(Qso.ID.EQ(String(strings.TrimSpace(id))))
	result, err := statement.ExecContext(ctx, s.db)
	if err != nil {
		return fmt.Errorf("delete QSO: %w", err)
	}

	changed, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("read deleted QSO count: %w", err)
	}
	if changed == 0 {
		return ErrNotFound
	}

	return nil
}

func (s *sqliteStore) MarkQRZSynced(
	ctx context.Context,
	id string,
	logID int64,
	syncedAt time.Time,
) (QSO, error) {
	id = strings.TrimSpace(id)
	if id == "" {
		return QSO{}, errors.New("QSO id is required")
	}
	if syncedAt.IsZero() {
		return QSO{}, errors.New("QRZ sync time is required")
	}
	if logID <= 0 {
		return QSO{}, errors.New("QRZ log id must be positive")
	}

	syncedAt = persistedTime(syncedAt)
	updatedAt := currentTime()
	statement := Qso.UPDATE(
		Qso.QrzSyncedAtUnixMs,
		Qso.QrzLogID,
		Qso.UpdatedAtUnixMs,
	).MODEL(dbmodel.Qso{
		QrzSyncedAtUnixMs: optionalTime(optional.Some(syncedAt)),
		QrzLogID:          &logID,
		UpdatedAtUnixMs:   updatedAt.UnixMilli(),
	}).WHERE(Qso.ID.EQ(String(id)))

	result, err := statement.ExecContext(ctx, s.db)
	if err != nil {
		return QSO{}, fmt.Errorf("mark QSO as synced to QRZ: %w", err)
	}

	changed, err := result.RowsAffected()
	if err != nil {
		return QSO{}, fmt.Errorf("read synced QSO count: %w", err)
	}
	if changed == 0 {
		return QSO{}, ErrNotFound
	}

	return s.Get(ctx, id)
}

func qsoFromModel(stored dbmodel.Qso) QSO {
	qso := QSO{
		ID:               stored.ID,
		StationCallsign:  stored.StationCallsign,
		Callsign:         stored.Callsign,
		StartedAt:        time.UnixMilli(stored.StartedAtUnixMs).UTC(),
		Mode:             stored.Mode,
		Submode:          stored.Submode,
		RSTSent:          stored.RstSent,
		RSTReceived:      stored.RstReceived,
		ExchangeSent:     stored.ExchangeSent,
		ExchangeReceived: stored.ExchangeReceived,
		Name:             stored.Name,
		QTH:              stored.Qth,
		Notes:            stored.Notes,
		CreatedAt:        time.UnixMilli(stored.CreatedAtUnixMs).UTC(),
		UpdatedAt:        time.UnixMilli(stored.UpdatedAtUnixMs).UTC(),
	}

	if stored.FrequencyHz != nil {
		qso.FrequencyHz = optional.Some(*stored.FrequencyHz)
	}
	if stored.QrzSyncedAtUnixMs != nil {
		qso.QRZSyncedAt = optional.Some(
			time.UnixMilli(*stored.QrzSyncedAtUnixMs).UTC(),
		)
	}
	if stored.QrzLogID != nil {
		qso.QRZLogID = optional.Some(*stored.QrzLogID)
	}

	return qso
}

func qsoToModel(qso QSO) dbmodel.Qso {
	return dbmodel.Qso{
		ID:                qso.ID,
		StationCallsign:   qso.StationCallsign,
		Callsign:          qso.Callsign,
		StartedAtUnixMs:   qso.StartedAt.UnixMilli(),
		FrequencyHz:       optionalPointer(qso.FrequencyHz),
		Mode:              qso.Mode,
		Submode:           qso.Submode,
		RstSent:           qso.RSTSent,
		RstReceived:       qso.RSTReceived,
		ExchangeSent:      qso.ExchangeSent,
		ExchangeReceived:  qso.ExchangeReceived,
		Name:              qso.Name,
		Qth:               qso.QTH,
		Notes:             qso.Notes,
		QrzSyncedAtUnixMs: optionalTime(qso.QRZSyncedAt),
		QrzLogID:          optionalPointer(qso.QRZLogID),
		CreatedAtUnixMs:   qso.CreatedAt.UnixMilli(),
		UpdatedAtUnixMs:   qso.UpdatedAt.UnixMilli(),
	}
}

func optionalPointer[T any](value optional.Value[T]) *T {
	stored, present := value.Get()
	if !present {
		return nil
	}

	return &stored
}

func optionalTime(value optional.Value[time.Time]) *int64 {
	stored, present := value.Get()
	if !present {
		return nil
	}

	unixMilliseconds := persistedTime(stored).UnixMilli()
	return &unixMilliseconds
}

func newID() (string, error) {
	var bytes [16]byte
	if _, err := rand.Read(bytes[:]); err != nil {
		return "", fmt.Errorf("generate QSO id: %w", err)
	}

	return hex.EncodeToString(bytes[:]), nil
}
