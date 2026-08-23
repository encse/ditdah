//go:build screenshots

package tui

import (
	"time"

	domain "ditdah/internal/logbook"
	"ditdah/internal/optional"
	ui "ditdah/internal/tui"
	"ditdah/internal/tui/modal"
)

// NewScreenshotDialog creates the populated QSO editor shown in product images.
func NewScreenshotDialog(host ui.PageHost) modal.Dialog {
	return newQSOEditor(host, screenshotQSO(), nil, nil)
}

func screenshotQSO() domain.QSO {
	return domain.QSO{
		StationCallsign: "HA7NCS",
		Callsign:        "HA5LA",
		StartedAt: time.Date(
			2026,
			time.August,
			21,
			18,
			40,
			0,
			0,
			time.FixedZone("CEST", 2*60*60),
		),
		FrequencyHz:      optional.Some[int64](14_025_000),
		Mode:             "CW",
		RSTSent:          "599",
		RSTReceived:      "599",
		ExchangeSent:     "DAVID ERD",
		ExchangeReceived: "LACI BUDAPEST",
		Name:             `László "Laci" Áshin`,
		QTH:              "Budapest, Hungary",
		Notes:            "Relaxed evening QSO on 20 metres.",
	}
}
