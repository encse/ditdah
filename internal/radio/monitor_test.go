package radio

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	domain "ditdah/internal/settings"
	"ditdah/internal/syncutil"
)

func TestMonitorPollsConfiguredRadioAndPublishesFrequencyChanges(t *testing.T) {
	config := Config{
		ModelID:   3073,
		ModelName: "Icom IC-7300",
		Port:      "/dev/cu.radio",
		BaudRate:  19200,
	}
	store := newMonitorSettingsStore(settingsFromConfig(config))
	rig := &controlledRig{
		checks:  make(chan Config, 4),
		results: make(chan rigResult, 4),
	}
	service := newMonitor(store, rig, 10*time.Millisecond)
	changes := service.Subscribe()
	defer changes.Close()
	cancel, done := runMonitor(t, service)
	defer stopMonitor(t, cancel, done)

	assertRigCheck(t, rig, config)
	rig.results <- rigResult{frequency: 28_039_000}
	waitForRadioStatus(t, changes, service, Status{FrequencyHz: 28_039_000})

	assertRigCheck(t, rig, config)
	rig.results <- rigResult{frequency: 28_039_600}
	waitForRadioStatus(t, changes, service, Status{FrequencyHz: 28_039_600})
}

func TestMonitorReloadsRadioImmediatelyWhenSettingsChange(t *testing.T) {
	store := newMonitorSettingsStore(domain.Settings{})
	rig := &controlledRig{
		checks:  make(chan Config, 1),
		results: make(chan rigResult, 1),
	}
	service := newMonitor(store, rig, time.Hour)
	changes := service.Subscribe()
	defer changes.Close()
	cancel, done := runMonitor(t, service)
	defer stopMonitor(t, cancel, done)

	waitForRadioStatus(
		t,
		changes,
		service,
		Status{Error: "Radio is not configured"},
	)
	config := Config{
		ModelID:   1,
		ModelName: "Test Rig",
		Port:      "COM3",
		BaudRate:  9600,
	}
	if _, err := store.Save(t.Context(), settingsFromConfig(config)); err != nil {
		t.Fatal(err)
	}
	assertRigCheck(t, rig, config)
	rig.results <- rigResult{frequency: 7_030_000}
	waitForRadioStatus(t, changes, service, Status{FrequencyHz: 7_030_000})
}

func TestMonitorCancelsCurrentPollingSessionWhenSettingsChange(t *testing.T) {
	first := Config{ModelID: 1, Port: "COM3", BaudRate: 9600}
	second := Config{ModelID: 2, Port: "COM4", BaudRate: 19200}
	store := newMonitorSettingsStore(settingsFromConfig(first))
	rig := &controlledRig{
		checks:  make(chan Config, 2),
		results: make(chan rigResult, 1),
	}
	service := newMonitor(store, rig, time.Hour)
	changes := service.Subscribe()
	defer changes.Close()
	cancel, done := runMonitor(t, service)
	defer stopMonitor(t, cancel, done)

	assertRigCheck(t, rig, first)
	if _, err := store.Save(t.Context(), settingsFromConfig(second)); err != nil {
		t.Fatal(err)
	}
	assertRigCheck(t, rig, second)
	rig.results <- rigResult{frequency: 14_074_000}
	waitForRadioStatus(t, changes, service, Status{FrequencyHz: 14_074_000})
}

func TestMonitorPublishesRadioErrors(t *testing.T) {
	config := Config{ModelID: 1, Port: "COM3", BaudRate: 9600}
	store := newMonitorSettingsStore(settingsFromConfig(config))
	rig := &controlledRig{
		checks:  make(chan Config, 1),
		results: make(chan rigResult, 1),
	}
	service := newMonitor(store, rig, time.Hour)
	changes := service.Subscribe()
	defer changes.Close()
	cancel, done := runMonitor(t, service)
	defer stopMonitor(t, cancel, done)

	assertRigCheck(t, rig, config)
	rig.results <- rigResult{err: errors.New("radio did not answer")}
	waitForRadioStatus(
		t,
		changes,
		service,
		Status{Error: "radio did not answer"},
	)
}

func runMonitor(t *testing.T, service *monitor) (context.CancelFunc, <-chan struct{}) {
	t.Helper()
	ctx, cancel := context.WithCancel(t.Context())
	done := make(chan struct{})
	go func() {
		service.Run(ctx)
		close(done)
	}()
	return cancel, done
}

func stopMonitor(t *testing.T, cancel context.CancelFunc, done <-chan struct{}) {
	t.Helper()
	cancel()
	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("radio monitor did not stop after cancellation")
	}
}

func assertRigCheck(t *testing.T, rig *controlledRig, want Config) {
	t.Helper()
	select {
	case got := <-rig.checks:
		if got != want {
			t.Fatalf("checked config = %#v, want %#v", got, want)
		}
	case <-time.After(time.Second):
		t.Fatal("radio was not checked")
	}
}

func waitForRadioStatus(
	t *testing.T,
	changes syncutil.Subscription,
	service *monitor,
	want Status,
) {
	t.Helper()
	ctx, cancel := context.WithTimeout(t.Context(), time.Second)
	defer cancel()
	for service.Status() != want {
		if err := changes.Wait(ctx); err != nil {
			t.Fatalf("wait for radio status %#v: %v", want, err)
		}
	}
}

type rigResult struct {
	frequency uint64
	err       error
}

type controlledRig struct {
	checks  chan Config
	results chan rigResult
}

func (*controlledRig) Models() ([]Model, error) { return nil, nil }

func (*controlledRig) Ports() ([]string, error) { return nil, nil }

func (r *controlledRig) Check(ctx context.Context, config Config) (uint64, error) {
	select {
	case <-ctx.Done():
		return 0, ctx.Err()
	case r.checks <- config:
	}
	select {
	case <-ctx.Done():
		return 0, ctx.Err()
	case result := <-r.results:
		return result.frequency, result.err
	}
}

type monitorSettingsStore struct {
	mu      sync.Mutex
	values  domain.Settings
	changes syncutil.Broadcaster
}

func newMonitorSettingsStore(values domain.Settings) *monitorSettingsStore {
	return &monitorSettingsStore{
		values:  values,
		changes: syncutil.NewBroadcaster(),
	}
}

func (s *monitorSettingsStore) Load(context.Context) (domain.Settings, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.values, nil
}

func (s *monitorSettingsStore) Save(
	_ context.Context,
	values domain.Settings,
) (domain.Settings, error) {
	s.mu.Lock()
	s.values = values
	s.mu.Unlock()
	s.changes.Activate()
	return values, nil
}

func (s *monitorSettingsStore) Subscribe() syncutil.Subscription {
	return s.changes.Subscribe()
}

func settingsFromConfig(config Config) domain.Settings {
	return domain.Settings{
		RadioModelID:    config.ModelID,
		RadioModelName:  config.ModelName,
		RadioSerialPort: config.Port,
		RadioBaudRate:   config.BaudRate,
	}
}
