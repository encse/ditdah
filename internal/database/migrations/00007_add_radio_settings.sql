-- +goose Up

ALTER TABLE application_settings
    ADD COLUMN radio_model_id INTEGER NOT NULL DEFAULT 0;
ALTER TABLE application_settings
    ADD COLUMN radio_model_name TEXT NOT NULL DEFAULT '';
ALTER TABLE application_settings
    ADD COLUMN radio_serial_port TEXT NOT NULL DEFAULT '';
ALTER TABLE application_settings
    ADD COLUMN radio_baud_rate INTEGER NOT NULL DEFAULT 0;

-- +goose Down

ALTER TABLE application_settings DROP COLUMN radio_baud_rate;
ALTER TABLE application_settings DROP COLUMN radio_serial_port;
ALTER TABLE application_settings DROP COLUMN radio_model_name;
ALTER TABLE application_settings DROP COLUMN radio_model_id;
