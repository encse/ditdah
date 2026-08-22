package qrz

import (
	"context"
	"testing"
	"time"

	domain "ditdah/internal/logbook"
	"ditdah/internal/optional"
	"ditdah/internal/settings"
)

func TestSynchronizerUploadsOnlyPendingAndReplacesOldLogID(t *testing.T) {
	syncedAt := time.Date(2026, 8, 18, 12, 0, 0, 0, time.UTC)
	store := &syncTestStore{qsos: []domain.QSO{
		{ID: "new", Callsign: "DL1ABC"},
		{
			ID:       "edited",
			Callsign: "OE1XYZ",
			QRZLogID: optional.Some[int64](41),
		},
		{
			ID:          "done",
			Callsign:    "F1ABC",
			QRZSyncedAt: optional.Some(syncedAt),
		},
	}}
	remote := &syncTestRemote{logIDs: []int64{40, 42}}
	syncer := NewSynchronizer(
		remote,
		store,
		syncTestSettings{values: settings.Settings{QRZAPIKey: "api-key"}},
	)

	count, err := syncer.Sync(t.Context())
	if err != nil {
		t.Fatalf("Sync() error = %v", err)
	}
	if count != 2 {
		t.Fatalf("Sync() count = %d, want 2", count)
	}
	if len(remote.uploaded) != 2 || remote.uploaded[0] != "new" ||
		remote.uploaded[1] != "edited" {
		t.Fatalf("uploaded = %#v", remote.uploaded)
	}
	if len(remote.deleted) != 1 || remote.deleted[0] != 41 {
		t.Fatalf("deleted = %#v, want [41]", remote.deleted)
	}
	if len(store.marked) != 2 || store.marked["new"] != 40 ||
		store.marked["edited"] != 42 {
		t.Fatalf("marked = %#v", store.marked)
	}
}

func TestSynchronizerDoesNotRequireAPIKeyWithoutPendingQSOs(t *testing.T) {
	store := &syncTestStore{qsos: []domain.QSO{{
		ID:          "done",
		QRZSyncedAt: optional.Some(time.Now()),
	}}}
	count, err := NewSynchronizer(
		&syncTestRemote{},
		store,
		syncTestSettings{},
	).Sync(t.Context())
	if err != nil || count != 0 {
		t.Fatalf("Sync() = %d, %v; want 0, nil", count, err)
	}
}

type syncTestStore struct {
	domain.Store
	qsos   []domain.QSO
	marked map[string]int64
}

func (s *syncTestStore) List(
	context.Context,
	domain.Filter,
) ([]domain.QSO, error) {
	return s.qsos, nil
}

func (s *syncTestStore) MarkQRZSynced(
	_ context.Context,
	id string,
	logID int64,
	_ time.Time,
) (domain.QSO, error) {
	if s.marked == nil {
		s.marked = make(map[string]int64)
	}
	s.marked[id] = logID
	return domain.QSO{ID: id}, nil
}

type syncTestSettings struct {
	settings.Store
	values settings.Settings
}

func (s syncTestSettings) Load(context.Context) (settings.Settings, error) {
	return s.values, nil
}

type syncTestRemote struct {
	logIDs   []int64
	uploaded []string
	deleted  []int64
}

func (r *syncTestRemote) UploadQSO(
	_ context.Context,
	_ string,
	qso domain.QSO,
) (int64, error) {
	r.uploaded = append(r.uploaded, qso.ID)
	logID := r.logIDs[0]
	r.logIDs = r.logIDs[1:]
	return logID, nil
}

func (r *syncTestRemote) DeleteQSO(
	_ context.Context,
	_ string,
	logID int64,
) error {
	r.deleted = append(r.deleted, logID)
	return nil
}
