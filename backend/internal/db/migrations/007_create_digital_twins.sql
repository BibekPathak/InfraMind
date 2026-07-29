-- +goose Up
CREATE TABLE digital_twins (
    asset_id UUID PRIMARY KEY REFERENCES assets(id) ON DELETE CASCADE,
    device_id UUID REFERENCES devices(id) ON DELETE SET NULL,
    metadata JSONB DEFAULT '{}',
    live_state JSONB DEFAULT '{}',
    maintenance_history JSONB DEFAULT '[]'::jsonb,
    ai_summary TEXT DEFAULT '',
    health_score DOUBLE PRECISION,
    health_level TEXT,
    synced_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_digital_twins_health_level ON digital_twins(health_level);
CREATE INDEX idx_digital_twins_synced_at ON digital_twins(synced_at DESC);

-- +goose Down
DROP TABLE IF EXISTS digital_twins;
