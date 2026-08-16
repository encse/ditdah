// Package settings contains application settings and their persistence boundary.
package settings

import (
	"context"
	"strings"
)

// Settings contains the user-configurable station and QRZ values.
type Settings struct {
	StationCallsign string
	QRZPassword     string
	QRZAPIKey       string
}

// Store loads and saves the application's single settings record.
type Store interface {
	Load(ctx context.Context) (Settings, error)
	Save(ctx context.Context, settings Settings) (Settings, error)
}

func normalize(settings Settings) Settings {
	settings.StationCallsign = strings.ToUpper(
		strings.TrimSpace(settings.StationCallsign),
	)
	settings.QRZAPIKey = strings.TrimSpace(settings.QRZAPIKey)
	return settings
}
