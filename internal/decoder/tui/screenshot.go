//go:build screenshots

package tui

import (
	"ditdah/internal/callsign"
	"ditdah/internal/optional"
	ui "ditdah/internal/tui"
)

// NewScreenshotPage creates the populated decoder shown in product images.
func NewScreenshotPage(host ui.PageHost) ui.Page {
	page := newPage(host, nil, nil, nil, nil, nil, nil)
	page.audioStatus = "Listening: USB Audio CODEC"
	page.radioStatus = "14.025 MHz"
	page.callsigns = []string{"HA5LA", "DL1ABC", "G3XYZ", "I2RTF"}
	page.selectedCallsign = "HA5LA"
	page.renderCallsigns()
	_, _ = page.decodedText.WriteString(
		"CQ CQ CQ DE HA5LA HA5LA K\n\n" +
			"HA5LA DE HA7NCS HA7NCS K\n\n" +
			"HA7NCS DE HA5LA GM ES TNX FER CALL\n" +
			"UR RST 579 579 NAME LACI QTH BUDAPEST\n" +
			"WX SUNNY TEMP 22C HW CPY? HA7NCS DE HA5LA K\n\n" +
			"HA5LA DE HA7NCS FB LACI TNX 579\n" +
			"NAME DAVID QTH ERD\n",
	)
	page.renderDecodedText()
	page.details.setEntry(callsign.Entry{
		Status: callsign.StatusReady,
		Record: optional.Some(callsign.Record{
			Callsign: "HA5LA",
			Name:     optional.Some("László Áshin"),
			Nickname: optional.Some("Laci"),
			QTH:      optional.Some("Budapest"),
			Country:  optional.Some("Hungary"),
			Grid:     optional.Some("JN97mm"),
			CQZone:   optional.Some("15"),
			ITUZone:  optional.Some("28"),
		}),
	})
	return page
}
