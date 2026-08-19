package qrz

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	domain "morsemanual/internal/logbook"
	"morsemanual/internal/settings"
)

const syncPageSize = 500

// Synchronizer uploads locally pending QSOs to one QRZ logbook.
type Synchronizer interface {
	Sync(ctx context.Context) (int, error)
}

type synchronizer struct {
	remote   Logbook
	logbook  domain.Store
	settings settings.Store
}

// NewSynchronizer creates the QRZ synchronization application service.
func NewSynchronizer(
	remote Logbook,
	logbook domain.Store,
	settingsStore settings.Store,
) Synchronizer {
	return &synchronizer{
		remote:   remote,
		logbook:  logbook,
		settings: settingsStore,
	}
}

func (s *synchronizer) Sync(ctx context.Context) (int, error) {
	pending, err := s.pendingQSOs(ctx)
	if err != nil {
		return 0, err
	}
	if len(pending) == 0 {
		return 0, nil
	}
	configured, err := s.settings.Load(ctx)
	if err != nil {
		return 0, fmt.Errorf("load QRZ.com settings: %w", err)
	}
	apiKey := strings.TrimSpace(configured.QRZAPIKey)
	if apiKey == "" {
		return 0, errors.New("configure a QRZ.com Logbook API key in Settings first")
	}

	synced := 0
	for _, qso := range pending {
		logID, err := s.remote.UploadQSO(ctx, apiKey, qso)
		if err != nil {
			return synced, fmt.Errorf("upload QSO with %s: %w", qso.Callsign, err)
		}
		if oldLogID, present := qso.QRZLogID.Get(); present && oldLogID != logID {
			if err := s.remote.DeleteQSO(ctx, apiKey, oldLogID); err != nil {
				return synced, fmt.Errorf(
					"remove replaced QRZ.com QSO with %s: %w",
					qso.Callsign,
					err,
				)
			}
		}
		if _, err := s.logbook.MarkQRZSynced(ctx, qso.ID, logID, time.Now()); err != nil {
			return synced, fmt.Errorf("store QRZ.com sync for %s: %w", qso.Callsign, err)
		}
		synced++
	}
	return synced, nil
}

func (s *synchronizer) pendingQSOs(ctx context.Context) ([]domain.QSO, error) {
	var pending []domain.QSO
	for offset := 0; ; offset += syncPageSize {
		qsos, err := s.logbook.List(ctx, domain.Filter{
			Limit:  syncPageSize,
			Offset: offset,
		})
		if err != nil {
			return nil, fmt.Errorf("load QSOs for QRZ.com sync: %w", err)
		}
		for _, qso := range qsos {
			if !qso.QRZSyncedAt.IsSome() {
				pending = append(pending, qso)
			}
		}
		if len(qsos) < syncPageSize {
			return pending, nil
		}
	}
}
