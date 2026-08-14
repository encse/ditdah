-- +goose Up

CREATE TABLE IF NOT EXISTS qso (
	id                    TEXT PRIMARY KEY NOT NULL,
	station_callsign      TEXT NOT NULL,
	callsign              TEXT NOT NULL,
	started_at_unix_ms    INTEGER NOT NULL,
	frequency_hz          INTEGER,
	mode                  TEXT NOT NULL,
	submode               TEXT NOT NULL DEFAULT '',
	rst_sent              TEXT NOT NULL DEFAULT '',
	rst_received          TEXT NOT NULL DEFAULT '',
	exchange_sent         TEXT NOT NULL DEFAULT '',
	exchange_received     TEXT NOT NULL DEFAULT '',
	name                  TEXT NOT NULL DEFAULT '',
	qth                   TEXT NOT NULL DEFAULT '',
	notes                 TEXT NOT NULL DEFAULT '',
	qrz_synced_at_unix_ms INTEGER,
	created_at_unix_ms    INTEGER NOT NULL,
	updated_at_unix_ms    INTEGER NOT NULL,
	CHECK (frequency_hz IS NULL OR frequency_hz > 0)
);

CREATE INDEX IF NOT EXISTS qso_started_at_idx
	ON qso(started_at_unix_ms DESC);
CREATE INDEX IF NOT EXISTS qso_callsign_idx
	ON qso(callsign);
CREATE INDEX IF NOT EXISTS qso_station_callsign_idx
	ON qso(station_callsign);

-- +goose Down

DROP TABLE IF EXISTS qso;
