-- +goose Up
CREATE TABLE telemetry (
    time TIMESTAMPTZ NOT NULL,
    device_id UUID NOT NULL,
    temperature DOUBLE PRECISION,
    current_amps DOUBLE PRECISION,
    voltage DOUBLE PRECISION,
    humidity DOUBLE PRECISION
);

SELECT create_hypertable('telemetry', 'time', chunk_time_interval => INTERVAL '1 day');

CREATE INDEX idx_telemetry_device_id ON telemetry(device_id);
CREATE INDEX idx_telemetry_time ON telemetry(time DESC);

-- +goose Down
DROP TABLE IF EXISTS telemetry;
