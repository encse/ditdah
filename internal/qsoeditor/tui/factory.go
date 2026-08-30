package tui

import (
	"context"
	"errors"
	"fmt"
	"math"
	"strings"
	"time"

	"ditdah/internal/callsign"
	domain "ditdah/internal/logbook"
	"ditdah/internal/optional"
	"ditdah/internal/radio"
	"ditdah/internal/settings"
	ui "ditdah/internal/tui"
)

// Factory opens QSO editors owned by a page.
type Factory interface {
	Create(owner ui.Owner, callsign string)
	Edit(owner ui.Owner, qso domain.QSO)
}

type factory struct {
	host     ui.PageHost
	store    domain.Store
	settings settings.Store
	lookup   callsign.Service
	radio    radio.StatusReader
}

// New creates a QSO editor factory.
func New(
	host ui.PageHost,
	store domain.Store,
	settingsStore settings.Store,
	lookup callsign.Service,
	radioStatus radio.StatusReader,
) Factory {
	return &factory{
		host:     host,
		store:    store,
		settings: settingsStore,
		lookup:   lookup,
		radio:    radioStatus,
	}
}

func (f *factory) Create(owner ui.Owner, contactedCallsign string) {
	qso := domain.QSO{
		Callsign:  strings.ToUpper(strings.TrimSpace(contactedCallsign)),
		StartedAt: time.Now(),
		Mode:      "CW",
	}
	if f.radio != nil {
		status := f.radio.Status()
		if status.Error == "" && status.FrequencyHz > 0 &&
			status.FrequencyHz <= math.MaxInt64 {
			qso.FrequencyHz = optional.Some(int64(status.FrequencyHz))
		}
	}
	f.host.Background(owner, func(ctx context.Context) {
		err := f.resolveDraft(ctx, &qso)
		if ctx.Err() != nil {
			return
		}
		f.host.Update(owner, func() {
			f.open(owner, qso, f.add, err)
		})
	})
}

func (f *factory) Edit(owner ui.Owner, qso domain.QSO) {
	f.open(owner, qso, f.update, nil)
}

func (f *factory) resolveDraft(ctx context.Context, qso *domain.QSO) error {
	var resultErr error
	if f.settings != nil {
		configured, err := f.settings.Load(ctx)
		if err != nil {
			resultErr = errors.Join(resultErr, fmt.Errorf("load settings: %w", err))
		} else {
			qso.StationCallsign = configured.StationCallsign
		}
	}
	if qso.Callsign == "" || f.lookup == nil {
		return resultErr
	}

	entry, err := f.lookup.Lookup(ctx, qso.Callsign)
	if err != nil {
		return errors.Join(resultErr, fmt.Errorf("look up %s: %w", qso.Callsign, err))
	}
	if record, present := entry.Record.Get(); present {
		qso.Name, _ = record.Name.Get()
		if qso.Name == "" {
			qso.Name, _ = record.Nickname.Get()
		}
		qso.QTH, _ = record.QTH.Get()
	}
	return resultErr
}

func (f *factory) open(
	owner ui.Owner,
	qso domain.QSO,
	save saveQSOFunc,
	err error,
) {
	editor := newQSOEditor(f.host, qso, save, f.lookup)
	if err != nil {
		editor.showError(err)
	}
	editor.setHandle(f.host.OpenModal(owner, editor))
}

func (f *factory) add(ctx context.Context, qso domain.QSO) (domain.QSO, error) {
	created, err := f.store.Add(ctx, qso)
	if err != nil {
		return domain.QSO{}, fmt.Errorf("add QSO: %w", err)
	}
	return created, nil
}

func (f *factory) update(ctx context.Context, qso domain.QSO) (domain.QSO, error) {
	updated, err := f.store.Update(ctx, qso)
	if err != nil {
		return domain.QSO{}, fmt.Errorf("update QSO: %w", err)
	}
	return updated, nil
}
