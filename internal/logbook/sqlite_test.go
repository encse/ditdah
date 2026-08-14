package logbook

import (
	"context"
	"errors"
	"testing"
	"time"

	"morsemanual/internal/database"
	"morsemanual/internal/optional"
)

func TestSQLiteStoreLifecycle(t *testing.T) {
	store := openTestStore(t)
	ctx := context.Background()
	startedAt := time.Date(2026, time.August, 13, 16, 17, 0, 0, time.UTC)

	created, err := store.Add(ctx, QSO{
		StationCallsign:  " ha7ncs ",
		Callsign:         " dl8eca/p ",
		StartedAt:        startedAt,
		FrequencyHz:      optional.Some[int64](7_023_500),
		Mode:             " cw ",
		RSTSent:          "559",
		RSTReceived:      "539",
		ExchangeReceived: " 123 ",
		Name:             "Flo",
		QTH:              "Remscheid",
	})
	if err != nil {
		t.Fatalf("Add() error = %v", err)
	}
	if created.ID == "" {
		t.Fatal("Add() returned an empty id")
	}
	if created.StationCallsign != "HA7NCS" || created.Callsign != "DL8ECA/P" {
		t.Fatalf("Add() callsigns = %q, %q", created.StationCallsign, created.Callsign)
	}

	loaded, err := store.Get(ctx, created.ID)
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}
	if loaded != created {
		t.Fatalf("Get() = %#v, want %#v", loaded, created)
	}

	loaded.Mode = "SSB"
	loaded.Submode = "USB"
	loaded.Notes = "portable contact"
	loaded.QRZSyncedAt = optional.Some(time.Now())
	updated, err := store.Update(ctx, loaded)
	if err != nil {
		t.Fatalf("Update() error = %v", err)
	}
	if updated.Mode != "SSB" || updated.Submode != "USB" {
		t.Fatalf("Update() mode = %q/%q", updated.Mode, updated.Submode)
	}
	if updated.QRZSyncedAt.IsSome() {
		t.Fatalf("Update() QRZSyncedAt = %v, want none", updated.QRZSyncedAt)
	}

	syncedAt := time.Date(2026, 8, 14, 12, 30, 0, 0, time.UTC)
	synced, err := store.MarkQRZSynced(ctx, created.ID, syncedAt)
	if err != nil {
		t.Fatalf("MarkQRZSynced() error = %v", err)
	}
	storedSyncedAt, present := synced.QRZSyncedAt.Get()
	if !present || !storedSyncedAt.Equal(syncedAt) {
		t.Fatalf("MarkQRZSynced() time = %v, %v; want %v, true", storedSyncedAt, present, syncedAt)
	}

	if err := store.Delete(ctx, created.ID); err != nil {
		t.Fatalf("Delete() error = %v", err)
	}
	if _, err := store.Get(ctx, created.ID); !errors.Is(err, ErrNotFound) {
		t.Fatalf("Get() after Delete() error = %v, want ErrNotFound", err)
	}
}

func TestSQLiteStoreListAndSearch(t *testing.T) {
	store := openTestStore(t)
	ctx := context.Background()

	entries := []QSO{
		{
			StationCallsign: "HA7NCS",
			Callsign:        "SA6AUT",
			StartedAt:       time.Date(2026, 8, 11, 18, 30, 0, 0, time.UTC),
			FrequencyHz:     optional.Some[int64](14_041_990),
			Mode:            "CW",
			Name:            "Marcin",
			QTH:             "Stromstad",
		},
		{
			StationCallsign: "HA7NCS",
			Callsign:        "YP0C",
			StartedAt:       time.Date(2026, 8, 1, 19, 21, 0, 0, time.UTC),
			FrequencyHz:     optional.Some[int64](14_184_000),
			Mode:            "SSB",
			Notes:           "contest",
		},
	}

	for _, entry := range entries {
		if _, err := store.Add(ctx, entry); err != nil {
			t.Fatalf("Add() error = %v", err)
		}
	}

	all, err := store.List(ctx, Filter{})
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}
	if len(all) != 2 || all[0].Callsign != "SA6AUT" {
		t.Fatalf("List() = %#v", all)
	}

	found, err := store.List(ctx, Filter{Search: "contest"})
	if err != nil {
		t.Fatalf("List(search) error = %v", err)
	}
	if len(found) != 1 || found[0].Callsign != "YP0C" {
		t.Fatalf("List(search) = %#v", found)
	}
}

func TestSQLiteStoreRejectsInvalidQSO(t *testing.T) {
	store := openTestStore(t)

	_, err := store.Add(context.Background(), QSO{
		StationCallsign: "HA7NCS",
		StartedAt:       time.Now(),
		Mode:            "CW",
	})
	if err == nil {
		t.Fatal("Add() error = nil, want validation error")
	}
}

func TestSQLiteStorePreservesMissingFrequency(t *testing.T) {
	store := openTestStore(t)
	ctx := context.Background()

	created, err := store.Add(ctx, QSO{
		StationCallsign: "HA7NCS",
		Callsign:        "DL8ECA/P",
		StartedAt:       time.Date(2026, 8, 13, 16, 17, 0, 0, time.UTC),
		Mode:            "CW",
	})
	if err != nil {
		t.Fatalf("Add() error = %v", err)
	}

	loaded, err := store.Get(ctx, created.ID)
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}
	if loaded.FrequencyHz.IsSome() {
		t.Fatalf("Get() FrequencyHz = %v, want none", loaded.FrequencyHz)
	}
}

func openTestStore(t *testing.T) Store {
	t.Helper()

	db, err := database.OpenMemory()
	if err != nil {
		t.Fatalf("database.OpenMemory() error = %v", err)
	}
	t.Cleanup(func() {
		if err := db.Close(); err != nil {
			t.Errorf("database Close() error = %v", err)
		}
	})

	return NewSQLiteStore(db)
}
