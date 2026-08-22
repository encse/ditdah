// Package settings contains application settings and their persistence boundary.
package settings

import (
	"context"
	"strings"

	"morsemanual/internal/syncutil"
)

// Settings contains the user-configurable station, QRZ, and audio values.
type Settings struct {
	StationCallsign    string
	QRZPassword        string
	QRZAPIKey          string
	MorseInputDeviceID string
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
	return settings
}
