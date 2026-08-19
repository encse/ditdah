-- +goose Up

ALTER TABLE qso ADD COLUMN qrz_log_id INTEGER;

-- +goose Down

ALTER TABLE qso DROP COLUMN qrz_log_id;
