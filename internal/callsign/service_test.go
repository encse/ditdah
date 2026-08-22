package callsign

import (
	"context"
	"errors"
	"reflect"
	"testing"
	"time"

	"morsemanual/internal/optional"
	"morsemanual/internal/settings"
	"morsemanual/internal/syncutil"
)

func TestServiceReturnsCachedEntryWithoutLoadingCredentials(t *testing.T) {
	want := Entry{Callsign: "HA7NCS", Status: StatusReady}
	cache := &serviceTestCache{entry: want}
	credentials := &serviceTestSettings{loadErr: errors.New("must not load")}
	provider := &serviceTestProvider{err: errors.New("must not query")}
	service := newService(cache, credentials, provider, time.Now)

	got, err := service.Lookup(t.Context(), " ha7ncs ")
	if err != nil {
		t.Fatalf("Lookup() error = %v", err)
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("Lookup() = %#v, want %#v", got, want)
	}
	if credentials.loads != 0 || provider.calls != 0 || cache.saves != 0 {
		t.Fatalf(
			"loads = %d, provider calls = %d, saves = %d; want all zero",
			credentials.loads, provider.calls, cache.saves,
		)
	}
}

func TestServiceQueriesQRZAndCachesMiss(t *testing.T) {
	now := time.Date(2026, 8, 19, 12, 30, 0, 0, time.UTC)
	cache := &serviceTestCache{lookupErr: ErrNotFound}
	credentials := &serviceTestSettings{values: settings.Settings{
		StationCallsign: "HA7NCS",
		QRZPassword:     "secret",
	}}
	provider := &serviceTestProvider{record: Record{
		Callsign: "DL1DAW",
		Country:  optional.Some("Germany"),
	}}
	service := newService(cache, credentials, provider, func() time.Time {
		return now
	})

	got, err := service.Lookup(t.Context(), "sv8/dl1daw")
	if err != nil {
		t.Fatalf("Lookup() error = %v", err)
	}
	if provider.username != "HA7NCS" || provider.password != "secret" ||
		provider.callsign != "DL1DAW" {
		t.Fatalf("provider request = %#v", provider)
	}
	if got.Callsign != "SV8/DL1DAW" || got.QueryCallsign != "DL1DAW" ||
		got.Status != StatusReady || got.SavedAt != now ||
		!reflect.DeepEqual(got, cache.entry) {
		t.Fatalf("Lookup() = %#v", got)
	}
}

func TestServiceDoesNotQueryWithoutConfiguredLogin(t *testing.T) {
	cache := &serviceTestCache{lookupErr: ErrNotFound}
	provider := &serviceTestProvider{}
	service := newService(
		cache,
		&serviceTestSettings{},
		provider,
		time.Now,
	)

	_, err := service.Lookup(t.Context(), "DL1ABC")
	if !errors.Is(err, ErrCredentialsUnavailable) {
		t.Fatalf("Lookup() error = %v, want ErrCredentialsUnavailable", err)
	}
	if provider.calls != 0 || cache.saves != 0 {
		t.Fatalf("provider calls = %d, cache saves = %d; want zero", provider.calls, cache.saves)
	}
}

func TestServiceCachesProviderNotFound(t *testing.T) {
	cache := &serviceTestCache{lookupErr: ErrNotFound}
	provider := &serviceTestProvider{err: ErrProviderNotFound}
	service := newService(
		cache,
		&serviceTestSettings{values: settings.Settings{
			StationCallsign: "HA7NCS",
			QRZPassword:     "secret",
		}},
		provider,
		time.Now,
	)

	got, err := service.Lookup(t.Context(), "N0PE")
	if err != nil {
		t.Fatalf("Lookup() error = %v", err)
	}
	if got.Status != StatusError || got.Record.IsSome() || cache.saves != 1 {
		t.Fatalf("Lookup() = %#v, saves = %d", got, cache.saves)
	}
}

type serviceTestCache struct {
	entry     Entry
	lookupErr error
	saves     int
}

func (c *serviceTestCache) Lookup(context.Context, string) (Entry, error) {
	return c.entry, c.lookupErr
}

func (c *serviceTestCache) Save(_ context.Context, entry Entry) (Entry, error) {
	c.saves++
	c.entry = normalize(entry)
	return c.entry, nil
}

type serviceTestSettings struct {
	values  settings.Settings
	loadErr error
	loads   int
}

func (s *serviceTestSettings) Load(context.Context) (settings.Settings, error) {
	s.loads++
	return s.values, s.loadErr
}

func (s *serviceTestSettings) Save(
	context.Context,
	settings.Settings,
) (settings.Settings, error) {
	return settings.Settings{}, nil
}

func (s *serviceTestSettings) Subscribe() syncutil.Subscription {
	return syncutil.NewBroadcaster().Subscribe()
}

type serviceTestProvider struct {
	record   Record
	err      error
	calls    int
	username string
	password string
	callsign string
}

func (p *serviceTestProvider) LookupCallsign(
	_ context.Context,
	username string,
	password string,
	callsign string,
) (Record, error) {
	p.calls++
	p.username = username
	p.password = password
	p.callsign = callsign
	return p.record, p.err
}
