-- +goose Up

ALTER TABLE application_settings
    ADD COLUMN configured INTEGER NOT NULL DEFAULT 0;

UPDATE application_settings
SET configured = 1
WHERE station_callsign <> ''
   OR qrz_password <> ''
   OR qrz_api_key <> ''
   OR morse_input_device_id <> '';

-- +goose Down

ALTER TABLE application_settings DROP COLUMN configured;
