package settings

import (
	"context"
	"testing"
	"time"

	"ditdah/internal/database"
)

func TestSQLiteStoreLoadsEmptySettingsInitially(t *testing.T) {
	store := openTestStore(t)

	loaded, err := store.Load(context.Background())
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if loaded != (Settings{}) {
		t.Fatalf("Load() = %#v, want empty settings", loaded)
	}
}

func TestSQLiteStoreSavesAndLoadsSettings(t *testing.T) {
	store := openTestStore(t)
	ctx := context.Background()
	want := Settings{
		StationCallsign:    " ha7ncs ",
		QRZPassword:        " password with spaces ",
		QRZAPIKey:          " api-key ",
		MorseInputDeviceID: " input-id ",
	}

	saved, err := store.Save(ctx, want)
	if err != nil {
		t.Fatalf("Save() error = %v", err)
	}
	want.StationCallsign = "HA7NCS"
	want.QRZAPIKey = "api-key"
	want.MorseInputDeviceID = "input-id"
	want.Configured = true
	if saved != want {
		t.Fatalf("Save() = %#v, want %#v", saved, want)
	}

	loaded, err := store.Load(ctx)
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if loaded != want {
		t.Fatalf("Load() = %#v, want %#v", loaded, want)
	}
}

func TestSQLiteStoreOverwritesSettings(t *testing.T) {
	store := openTestStore(t)
	ctx := context.Background()
	if _, err := store.Save(ctx, Settings{StationCallsign: "HA7NCS"}); err != nil {
		t.Fatalf("first Save() error = %v", err)
	}
	want := Settings{
		StationCallsign:    "OE1ABC",
		QRZPassword:        "secret",
		QRZAPIKey:          "key",
		MorseInputDeviceID: "capture-2",
		Configured:         true,
	}
	if _, err := store.Save(ctx, want); err != nil {
		t.Fatalf("second Save() error = %v", err)
	}

	loaded, err := store.Load(ctx)
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if loaded != want {
		t.Fatalf("Load() = %#v, want %#v", loaded, want)
	}
}

func TestSQLiteStoreNotifiesSubscribersAfterSave(t *testing.T) {
	store := openTestStore(t)
	subscription := store.Subscribe()
	defer subscription.Close()

	if _, err := store.Save(t.Context(), Settings{
		StationCallsign: "HA7NCS",
	}); err != nil {
		t.Fatal(err)
	}

	ctx, cancel := context.WithTimeout(t.Context(), time.Second)
	defer cancel()
	if err := subscription.Wait(ctx); err != nil {
		t.Fatalf("Wait() after Save() = %v", err)
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
