package callsign

import (
	"context"
	"errors"
	"reflect"
	"testing"
	"time"

	"morsemanual/internal/database"
	"morsemanual/internal/optional"
)

func TestSQLiteStoreSavesPositiveAndNegativeLookups(t *testing.T) {
	db, err := database.OpenMemory()
	if err != nil {
		t.Fatalf("database.OpenMemory() error = %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })
	store := NewSQLiteStore(db)
	ctx := context.Background()

	ready, err := store.Save(ctx, Entry{
		Callsign:         " 4x6fr ",
		QueryCallsign:    " 4x6fr ",
		Status:           StatusReady,
		SavedAt:          time.Date(2026, 8, 8, 10, 42, 38, 922053800, time.UTC),
		ProvidersChecked: []string{"qrz.com", "qrz.com"},
		Record: optional.Some(Record{
			Callsign: "4X6FR",
			Country:  optional.Some("Israel"),
			Grid:     optional.Some("KM72kg"),
			Name:     optional.Some("ZVI STESSEL"),
			QRZURL:   optional.Some("https://www.qrz.com/db/4X6FR"),
			QTH:      optional.Some("HERZLIYYA 46684"),
		}),
	})
	if err != nil {
		t.Fatalf("Save(ready) error = %v", err)
	}
	if ready.Callsign != "4X6FR" || len(ready.ProvidersChecked) != 1 {
		t.Fatalf("Save(ready) = %#v", ready)
	}

	loaded, err := store.Lookup(ctx, " 4x6fr ")
	if err != nil {
		t.Fatalf("Lookup() error = %v", err)
	}
	if !reflect.DeepEqual(loaded, ready) {
		t.Fatalf("Lookup() = %#v, want %#v", loaded, ready)
	}

	failed, err := store.Save(ctx, Entry{
		Callsign:         "4X6F",
		QueryCallsign:    "4X6F",
		Status:           StatusError,
		SavedAt:          time.Date(2026, 8, 8, 10, 44, 54, 220625900, time.UTC),
		Error:            "Not found: 4X6F",
		ProvidersChecked: []string{"qrz.com"},
	})
	if err != nil {
		t.Fatalf("Save(error) error = %v", err)
	}
	loaded, err = store.Lookup(ctx, "4x6f")
	if err != nil {
		t.Fatalf("Lookup(error) error = %v", err)
	}
	if !reflect.DeepEqual(loaded, failed) || loaded.Record.IsSome() {
		t.Fatalf("Lookup(error) = %#v, want %#v", loaded, failed)
	}
}

func TestSQLiteStoreReplacesCachedLookup(t *testing.T) {
	db, err := database.OpenMemory()
	if err != nil {
		t.Fatalf("database.OpenMemory() error = %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })
	store := NewSQLiteStore(db)
	ctx := context.Background()

	_, err = store.Save(ctx, Entry{
		Callsign: "HA7NCS",
		Status:   StatusError,
		SavedAt:  time.Now(),
		Error:    "not found",
	})
	if err != nil {
		t.Fatalf("Save(error) error = %v", err)
	}
	want, err := store.Save(ctx, Entry{
		Callsign: "HA7NCS",
		Status:   StatusReady,
		SavedAt:  time.Now().Add(time.Minute),
		Record: optional.Some(Record{
			Callsign: "HA7NCS",
		}),
	})
	if err != nil {
		t.Fatalf("Save(ready) error = %v", err)
	}
	got, err := store.Lookup(ctx, "HA7NCS")
	if err != nil {
		t.Fatalf("Lookup() error = %v", err)
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("Lookup() = %#v, want %#v", got, want)
	}
}

func TestSQLiteStoreKeepsPortableCallsignAsSeparateCacheKey(t *testing.T) {
	db, err := database.OpenMemory()
	if err != nil {
		t.Fatalf("database.OpenMemory() error = %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })
	store := NewSQLiteStore(db)
	ctx := context.Background()

	for _, cacheCallsign := range []string{"DL1DAW", "SV8/DL1DAW"} {
		_, err := store.Save(ctx, Entry{
			Callsign:      cacheCallsign,
			QueryCallsign: "DL1DAW",
			Status:        StatusReady,
			SavedAt:       time.Now(),
			Record: optional.Some(Record{
				Callsign: "DL1DAW",
			}),
		})
		if err != nil {
			t.Fatalf("Save(%q) error = %v", cacheCallsign, err)
		}
	}

	portable, err := store.Lookup(ctx, "sv8/dl1daw")
	if err != nil {
		t.Fatalf("Lookup(portable) error = %v", err)
	}
	if portable.Callsign != "SV8/DL1DAW" || portable.QueryCallsign != "DL1DAW" {
		t.Fatalf("Lookup(portable) = %#v", portable)
	}
}

func TestSQLiteStoreReturnsNotFound(t *testing.T) {
	db, err := database.OpenMemory()
	if err != nil {
		t.Fatalf("database.OpenMemory() error = %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })

	_, err = NewSQLiteStore(db).Lookup(context.Background(), "UNKNOWN")
	if !errors.Is(err, ErrNotFound) {
		t.Fatalf("Lookup() error = %v, want ErrNotFound", err)
	}
}
