-- +goose Up

ALTER TABLE application_settings
    ADD COLUMN morse_input_device_id TEXT NOT NULL DEFAULT '';

-- +goose Down

ALTER TABLE application_settings DROP COLUMN morse_input_device_id;
