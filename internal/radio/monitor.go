package radio

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	domain "ditdah/internal/settings"
	"ditdah/internal/syncutil"

	"golang.org/x/sync/errgroup"
)

const defaultPollInterval = time.Second

var (
	errSettingsChanged = errors.New("radio settings changed")
	errReloadSettings  = errors.New("retry loading radio settings")
)

type monitor struct {
	settings domain.Store
	rig      Service
	interval time.Duration
	changes  syncutil.Broadcaster

	mu     sync.RWMutex
	status Status
}

// NewMonitor constructs the application-lifetime radio monitor.
func NewMonitor(settings domain.Store, rig Service) Monitor {
	return newMonitor(settings, rig, defaultPollInterval)
}

func newMonitor(
	settings domain.Store,
	rig Service,
	interval time.Duration,
) *monitor {
	return &monitor{
		settings: settings,
		rig:      rig,
		interval: interval,
		changes:  syncutil.NewBroadcaster(),
		status: Status{
			Error: "Radio service is starting",
		},
	}
}

func (m *monitor) Models() ([]Model, error) { return m.rig.Models() }

func (m *monitor) Ports() ([]string, error) { return m.rig.Ports() }

func (m *monitor) Check(ctx context.Context, config Config) (uint64, error) {
	return m.rig.Check(ctx, config)
}

func (m *monitor) Status() Status {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.status
}

func (m *monitor) Subscribe() syncutil.Subscription {
	return m.changes.Subscribe()
}

// Run watches saved settings and polls the configured radio until ctx ends.
func (m *monitor) Run(ctx context.Context) {
	if m.settings == nil || m.rig == nil {
		m.publish(Status{Error: "Radio service is unavailable"})
		<-ctx.Done()
		return
	}

	settingsChanges := m.settings.Subscribe()
	defer settingsChanges.Close()
	for ctx.Err() == nil {
		err := m.runSession(ctx, settingsChanges)
		if ctx.Err() != nil {
			return
		}
		if errors.Is(err, errSettingsChanged) ||
			errors.Is(err, errReloadSettings) {
			continue
		}
		return
	}
}

func (m *monitor) runSession(
	ctx context.Context,
	settingsChanges syncutil.Subscription,
) error {
	values, err := m.settings.Load(ctx)
	if err != nil {
		if ctx.Err() != nil {
			return ctx.Err()
		}
		m.publish(Status{Error: fmt.Sprintf("Load radio settings: %v", err)})
		return m.waitForSettingsOrReload(ctx, settingsChanges)
	}
	config := configFromSettings(values)
	if config.ModelID == 0 {
		m.publish(Status{Error: "Radio is not configured"})
		if err := settingsChanges.Wait(ctx); err != nil {
			return err
		}
		return errSettingsChanged
	}

	group, sessionCtx := errgroup.WithContext(ctx)
	group.Go(func() error {
		m.poll(sessionCtx, config)
		return nil
	})
	group.Go(func() error {
		if err := settingsChanges.Wait(sessionCtx); err != nil {
			return err
		}
		return errSettingsChanged
	})
	return group.Wait()
}

func (m *monitor) poll(ctx context.Context, config Config) {
	for ctx.Err() == nil {
		frequency, err := m.rig.Check(ctx, config)
		if ctx.Err() != nil {
			return
		}
		if err != nil {
			m.publish(Status{Error: err.Error()})
		} else {
			m.publish(Status{FrequencyHz: frequency})
		}
		timer := time.NewTimer(m.interval)
		select {
		case <-ctx.Done():
			timer.Stop()
			return
		case <-timer.C:
		}
	}
}

func (m *monitor) waitForSettingsOrReload(
	ctx context.Context,
	settingsChanges syncutil.Subscription,
) error {
	group, waitCtx := errgroup.WithContext(ctx)
	group.Go(func() error {
		if err := settingsChanges.Wait(waitCtx); err != nil {
			return err
		}
		return errSettingsChanged
	})
	group.Go(func() error {
		timer := time.NewTimer(m.interval)
		defer timer.Stop()
		select {
		case <-waitCtx.Done():
			return waitCtx.Err()
		case <-timer.C:
			return errReloadSettings
		}
	})
	return group.Wait()
}

func (m *monitor) publish(status Status) {
	m.mu.Lock()
	changed := status != m.status
	m.status = status
	m.mu.Unlock()
	if changed {
		m.changes.Activate()
	}
}

func configFromSettings(values domain.Settings) Config {
	return Config{
		ModelID:   values.RadioModelID,
		ModelName: values.RadioModelName,
		Port:      values.RadioSerialPort,
		BaudRate:  values.RadioBaudRate,
	}
}
