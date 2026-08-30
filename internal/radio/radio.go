// Package radio exposes the application's read-only radio boundary.
package radio

import (
	"context"

	"ditdah/internal/syncutil"
)

// Config identifies the Hamlib backend and serial connection used by a radio.
type Config struct {
	ModelID   int
	ModelName string
	Port      string
	BaudRate  int
}

// Model is a serial radio backend supported by Hamlib.
type Model struct {
	ID              int
	Manufacturer    string
	Name            string
	DefaultBaudRate int
}

// Status is the latest result produced by the application radio monitor.
// Exactly one of FrequencyHz and Error describes a useful result.
type Status struct {
	FrequencyHz uint64
	Error       string
}

// Service discovers radios and performs read-only connection checks.
type Service interface {
	Models() ([]Model, error)
	Ports() ([]string, error)
	Check(ctx context.Context, config Config) (frequencyHz uint64, err error)
}

// Monitor continuously reads the configured radio for the application
// lifetime and publishes coalesced status-change notifications.
type Monitor interface {
	Service
	Run(ctx context.Context)
	Status() Status
	Subscribe() syncutil.Subscription
}
