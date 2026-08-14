package logbook

import (
	"context"
	"errors"
	"strings"
	"time"
)

var ErrNotFound = errors.New("qso not found")

// QSO is one amateur-radio contact. Empty optional strings and a zero
// FrequencyHz mean that the corresponding value was not recorded.
type QSO struct {
	ID               string
	StationCallsign  string
	Callsign         string
	StartedAt        time.Time
	FrequencyHz      int64
	Mode             string
	Submode          string
	RSTSent          string
	RSTReceived      string
	ExchangeSent     string
	ExchangeReceived string
	Name             string
	QTH              string
	Notes            string
	QRZSyncedAt      time.Time
	CreatedAt        time.Time
	UpdatedAt        time.Time
}

// Filter limits and paginates QSOs returned by Store.List. Search matches the
// callsigns, name, QTH, mode, and notes. A zero Limit uses a sensible default.
type Filter struct {
	Search string
	Limit  int
	Offset int
}

// Store persists QSOs without exposing its concrete database implementation.
// Domain values cross this boundary by value.
type Store interface {
	Add(ctx context.Context, qso QSO) (QSO, error)
	Get(ctx context.Context, id string) (QSO, error)
	List(ctx context.Context, filter Filter) ([]QSO, error)
	Update(ctx context.Context, qso QSO) (QSO, error)
	Delete(ctx context.Context, id string) error
	MarkQRZSynced(ctx context.Context, id string, syncedAt time.Time) (QSO, error)
}

func normalizeQSO(qso QSO) QSO {
	qso.ID = strings.TrimSpace(qso.ID)
	qso.StationCallsign = strings.ToUpper(strings.TrimSpace(qso.StationCallsign))
	qso.Callsign = strings.ToUpper(strings.TrimSpace(qso.Callsign))
	qso.Mode = strings.ToUpper(strings.TrimSpace(qso.Mode))
	qso.Submode = strings.ToUpper(strings.TrimSpace(qso.Submode))
	qso.RSTSent = strings.TrimSpace(qso.RSTSent)
	qso.RSTReceived = strings.TrimSpace(qso.RSTReceived)
	qso.ExchangeSent = strings.TrimSpace(qso.ExchangeSent)
	qso.ExchangeReceived = strings.TrimSpace(qso.ExchangeReceived)
	qso.Name = strings.TrimSpace(qso.Name)
	qso.QTH = strings.TrimSpace(qso.QTH)
	qso.Notes = strings.TrimSpace(qso.Notes)

	if !qso.StartedAt.IsZero() {
		qso.StartedAt = persistedTime(qso.StartedAt)
	}
	if !qso.QRZSyncedAt.IsZero() {
		qso.QRZSyncedAt = persistedTime(qso.QRZSyncedAt)
	}

	return qso
}

func validateQSO(qso QSO) error {
	if qso.StationCallsign == "" {
		return errors.New("station callsign is required")
	}
	if qso.Callsign == "" {
		return errors.New("contacted callsign is required")
	}
	if qso.StartedAt.IsZero() {
		return errors.New("QSO start time is required")
	}
	if qso.FrequencyHz < 0 {
		return errors.New("frequency cannot be negative")
	}
	if qso.Mode == "" {
		return errors.New("mode is required")
	}
	return nil
}

func currentTime() time.Time {
	return persistedTime(time.Now())
}

func persistedTime(value time.Time) time.Time {
	return value.UTC().Truncate(time.Millisecond)
}
