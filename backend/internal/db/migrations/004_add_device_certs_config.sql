-- +goose Up
ALTER TABLE devices
    ADD COLUMN certificates JSONB DEFAULT '{}',
    ADD COLUMN config JSONB DEFAULT '{}',
    ADD COLUMN mqtt_username TEXT,
    ADD COLUMN mqtt_password TEXT;

-- +goose Down
ALTER TABLE devices
    DROP COLUMN IF EXISTS certificates,
    DROP COLUMN IF EXISTS config,
    DROP COLUMN IF EXISTS mqtt_username,
    DROP COLUMN IF EXISTS mqtt_password;
