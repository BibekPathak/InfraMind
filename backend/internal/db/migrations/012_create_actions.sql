-- +goose Up
CREATE TABLE actions (
    id UUID PRIMARY KEY,
    asset_id UUID NOT NULL REFERENCES assets(id) ON DELETE CASCADE,
    device_id UUID REFERENCES devices(id) ON DELETE CASCADE,
    type TEXT NOT NULL CHECK (type IN ('command', 'restart', 'config_change', 'notification')),
    payload JSONB NOT NULL DEFAULT '{}'::jsonb,
    source TEXT NOT NULL DEFAULT 'manual' CHECK (source IN ('ai', 'manual')),
    status TEXT NOT NULL DEFAULT 'proposed'
        CHECK (status IN ('proposed', 'approved', 'rejected', 'executed', 'failed')),
    approval_required BOOLEAN NOT NULL DEFAULT TRUE,
    auto_approved BOOLEAN NOT NULL DEFAULT FALSE,
    reason TEXT NOT NULL DEFAULT '',
    result TEXT,
    proposed_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    executed_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_actions_asset_id ON actions(asset_id);
CREATE INDEX idx_actions_status ON actions(status);
CREATE INDEX idx_actions_type ON actions(type);
CREATE INDEX idx_actions_created_at ON actions(created_at DESC);

-- +goose Down
DROP TABLE IF EXISTS actions;
