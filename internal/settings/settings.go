// Package settings contains application settings and their persistence boundary.
package settings

import (
	"context"
	"strings"

	"ditdah/internal/syncutil"
)

// Settings contains the user-configurable station, QRZ, audio, and radio values.
type Settings struct {
	StationCallsign    string
	QRZPassword        string
	QRZAPIKey          string
	MorseInputDeviceID string
	RadioModelID       int
	RadioModelName     string
	RadioSerialPort    string
	RadioBaudRate      int
	Configured         bool
}

// Store loads and saves the application's single settings record.
type Store interface {
	Load(ctx context.Context) (Settings, error)
	Save(ctx context.Context, settings Settings) (Settings, error)
	Subscribe() syncutil.Subscription
}

func normalize(settings Settings) Settings {
	settings.StationCallsign = strings.ToUpper(
		strings.TrimSpace(settings.StationCallsign),
	)
	settings.QRZAPIKey = strings.TrimSpace(settings.QRZAPIKey)
	settings.MorseInputDeviceID = strings.TrimSpace(settings.MorseInputDeviceID)
	settings.RadioModelName = strings.TrimSpace(settings.RadioModelName)
	settings.RadioSerialPort = strings.TrimSpace(settings.RadioSerialPort)
	return settings
}
