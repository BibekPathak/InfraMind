-- +goose Up
CREATE TABLE asset_types (
    type TEXT PRIMARY KEY,
    display_name TEXT NOT NULL,
    metrics JSONB NOT NULL DEFAULT '[]'::jsonb,
    thresholds JSONB NOT NULL DEFAULT '{}'::jsonb,
    health_weights JSONB NOT NULL DEFAULT '{}'::jsonb,
    active BOOLEAN NOT NULL DEFAULT TRUE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- FK from assets to asset_types is added in 010_seed_asset_types.sql
-- (after seed data exists, so existing asset rows validate correctly)

-- +goose Down
DROP TABLE IF EXISTS asset_types;
