-- +goose Up

CREATE TABLE callsign_lookup (
	callsign             TEXT PRIMARY KEY NOT NULL,
	query_callsign       TEXT NOT NULL,
	status               TEXT NOT NULL,
	saved_at_unix_ms     INTEGER NOT NULL,
	error                TEXT NOT NULL DEFAULT '',
	providers_checked    TEXT NOT NULL DEFAULT '[]',
	record_callsign      TEXT,
	country              TEXT,
	cq_zone              TEXT,
	grid                 TEXT,
	itu_zone             TEXT,
	name                 TEXT,
	nickname             TEXT,
	qrz_url              TEXT,
	qth                  TEXT,
	state                TEXT
);

-- +goose Down

DROP TABLE IF EXISTS callsign_lookup;
