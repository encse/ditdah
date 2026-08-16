-- +goose Up

CREATE TABLE application_settings (
	id               INTEGER PRIMARY KEY NOT NULL DEFAULT 1,
	station_callsign TEXT NOT NULL DEFAULT '',
	qrz_password     TEXT NOT NULL DEFAULT '',
	qrz_api_key      TEXT NOT NULL DEFAULT '',
	CHECK (id = 1)
);

INSERT INTO application_settings (id) VALUES (1);

-- +goose Down

DROP TABLE IF EXISTS application_settings;
