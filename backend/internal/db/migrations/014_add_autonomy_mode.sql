-- +goose Up
ALTER TABLE assets
    ADD COLUMN autonomy_mode TEXT NOT NULL DEFAULT 'manual'
        CHECK (autonomy_mode IN ('manual', 'advisory', 'autonomous'));

CREATE INDEX idx_assets_autonomy_mode ON assets(autonomy_mode);

-- +goose Down
ALTER TABLE assets DROP COLUMN IF EXISTS autonomy_mode;
