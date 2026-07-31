-- +goose Up
ALTER TABLE telemetry
    ADD COLUMN flow_rate DOUBLE PRECISION,
    ADD COLUMN pressure DOUBLE PRECISION,
    ADD COLUMN vibration DOUBLE PRECISION,
    ADD COLUMN rpm DOUBLE PRECISION,
    ADD COLUMN output_power DOUBLE PRECISION;

-- +goose Down
ALTER TABLE telemetry
    DROP COLUMN IF EXISTS flow_rate,
    DROP COLUMN IF EXISTS pressure,
    DROP COLUMN IF EXISTS vibration,
    DROP COLUMN IF EXISTS rpm,
    DROP COLUMN IF EXISTS output_power;
