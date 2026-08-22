// Package callsign contains cached callsign lookup results and their
// persistence boundary.
package callsign

import (
	"context"
	"errors"
	"strings"
	"time"

	"ditdah/internal/optional"
)

const (
	StatusReady = "ready"
	StatusError = "error"
)

var ErrNotFound = errors.New("callsign lookup not found")

// Record contains the station information returned by a callsign provider.
type Record struct {
	Callsign string
	Country  optional.Value[string]
	CQZone   optional.Value[string]
	Grid     optional.Value[string]
	ITUZone  optional.Value[string]
	Name     optional.Value[string]
	Nickname optional.Value[string]
	QRZURL   optional.Value[string]
	QTH      optional.Value[string]
	State    optional.Value[string]
}

// Entry is one positive or negative cached callsign lookup.
type Entry struct {
	Callsign         string
	QueryCallsign    string
	Status           string
	SavedAt          time.Time
	Error            string
	ProvidersChecked []string
	Record           optional.Value[Record]
}

// Store reads and replaces cached lookups by callsign.
type Store interface {
	Lookup(ctx context.Context, callsign string) (Entry, error)
	Save(ctx context.Context, entry Entry) (Entry, error)
}

func normalize(entry Entry) Entry {
	entry.Callsign = normalizeCallsign(entry.Callsign)
	if entry.QueryCallsign == "" {
		entry.QueryCallsign = entry.Callsign
	}
	entry.QueryCallsign = normalizeCallsign(entry.QueryCallsign)
	entry.Status = strings.ToLower(strings.TrimSpace(entry.Status))
	entry.Error = strings.TrimSpace(entry.Error)
	entry.SavedAt = entry.SavedAt.UTC().Truncate(time.Millisecond)
	entry.ProvidersChecked = normalizeProviders(entry.ProvidersChecked)

	if record, present := entry.Record.Get(); present {
		record.Callsign = normalizeCallsign(record.Callsign)
		entry.Record = optional.Some(record)
	}
	return entry
}

func normalizeCallsign(callsign string) string {
	return strings.ToUpper(strings.TrimSpace(callsign))
}

func normalizeProviders(providers []string) []string {
	result := make([]string, 0, len(providers))
	seen := make(map[string]struct{}, len(providers))
	for _, provider := range providers {
		provider = strings.TrimSpace(provider)
		if provider == "" {
			continue
		}
		if _, exists := seen[provider]; exists {
			continue
		}
		seen[provider] = struct{}{}
		result = append(result, provider)
	}
	return result
}

func validate(entry Entry) error {
	if entry.Callsign == "" {
		return errors.New("callsign is required")
	}
	if entry.QueryCallsign == "" {
		return errors.New("provider query callsign is required")
	}
	if entry.Status == "" {
		return errors.New("lookup status is required")
	}
	if entry.SavedAt.IsZero() {
		return errors.New("lookup save time is required")
	}
	if entry.Status == StatusReady && entry.Record.IsNone() {
		return errors.New("ready lookup requires a record")
	}
	return nil
}
