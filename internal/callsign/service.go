package callsign

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"morsemanual/internal/optional"
	"morsemanual/internal/settings"
)

const qrzProvider = "qrz.com"

var (
	// ErrCredentialsUnavailable means a cache miss cannot be resolved because
	// QRZ.com login credentials have not been configured.
	ErrCredentialsUnavailable = errors.New("QRZ.com login is not configured")
	// ErrProviderNotFound lets a provider report a cacheable negative lookup.
	ErrProviderNotFound = errors.New("callsign not found by provider")
)

// Provider retrieves one callsign record from an external data source.
type Provider interface {
	LookupCallsign(
		ctx context.Context,
		username string,
		password string,
		callsign string,
	) (Record, error)
}

// Service resolves callsigns through the persistent cache and its provider.
type Service interface {
	Lookup(ctx context.Context, callsign string) (Entry, error)
}

type service struct {
	cache    Store
	settings settings.Store
	provider Provider
	now      func() time.Time
}

// NewService creates a cache-first callsign lookup service.
func NewService(
	cache Store,
	settingsStore settings.Store,
	provider Provider,
) Service {
	return newService(cache, settingsStore, provider, time.Now)
}

func newService(
	cache Store,
	settingsStore settings.Store,
	provider Provider,
	now func() time.Time,
) Service {
	return service{
		cache:    cache,
		settings: settingsStore,
		provider: provider,
		now:      now,
	}
}

func (s service) Lookup(ctx context.Context, value string) (Entry, error) {
	cacheCallsign := normalizeCallsign(value)
	if cacheCallsign == "" {
		return Entry{}, errors.New("callsign is required")
	}
	if s.cache == nil {
		return Entry{}, errors.New("callsign cache is unavailable")
	}

	entry, err := s.cache.Lookup(ctx, cacheCallsign)
	if err == nil {
		return entry, nil
	}
	if !errors.Is(err, ErrNotFound) {
		return Entry{}, err
	}
	if s.settings == nil || s.provider == nil {
		return Entry{}, ErrCredentialsUnavailable
	}

	configured, err := s.settings.Load(ctx)
	if err != nil {
		return Entry{}, fmt.Errorf("load QRZ.com credentials: %w", err)
	}
	username := normalizeCallsign(configured.StationCallsign)
	if username == "" || configured.QRZPassword == "" {
		return Entry{}, ErrCredentialsUnavailable
	}

	query := providerQueryCallsign(cacheCallsign)
	record, err := s.provider.LookupCallsign(
		ctx,
		username,
		configured.QRZPassword,
		query,
	)
	entry = Entry{
		Callsign:         cacheCallsign,
		QueryCallsign:    query,
		SavedAt:          s.now(),
		ProvidersChecked: []string{qrzProvider},
	}
	if errors.Is(err, ErrProviderNotFound) {
		entry.Status = StatusError
		entry.Error = err.Error()
	} else if err != nil {
		return Entry{}, fmt.Errorf("lookup %s on QRZ.com: %w", cacheCallsign, err)
	} else {
		entry.Status = StatusReady
		entry.Record = optional.Some(record)
	}

	saved, err := s.cache.Save(ctx, entry)
	if err != nil {
		return Entry{}, fmt.Errorf("cache callsign lookup: %w", err)
	}
	return saved, nil
}

func providerQueryCallsign(callsign string) string {
	parts := strings.Split(callsign, "/")
	best := callsign
	for _, part := range parts {
		if !containsLetterAndDigit(part) {
			continue
		}
		if best == callsign || len(part) > len(best) {
			best = part
		}
	}
	return best
}

func containsLetterAndDigit(value string) bool {
	letter, digit := false, false
	for _, character := range value {
		switch {
		case character >= 'A' && character <= 'Z':
			letter = true
		case character >= '0' && character <= '9':
			digit = true
		}
	}
	return letter && digit
}
