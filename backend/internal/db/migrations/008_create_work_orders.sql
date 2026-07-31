-- +goose Up
CREATE TABLE work_orders (
    id UUID PRIMARY KEY,
    asset_id UUID NOT NULL REFERENCES assets(id) ON DELETE CASCADE,
    alert_id UUID REFERENCES alerts(id) ON DELETE SET NULL,
    type TEXT NOT NULL DEFAULT 'inspection'
        CHECK (type IN ('inspection', 'repair', 'replacement', 'firmware_update', 'diagnostic')),
    priority TEXT NOT NULL DEFAULT 'medium'
        CHECK (priority IN ('low', 'medium', 'high', 'critical')),
    status TEXT NOT NULL DEFAULT 'open'
        CHECK (status IN ('open', 'assigned', 'in_progress', 'completed', 'cancelled')),
    assigned_to TEXT,
    estimated_cost DOUBLE PRECISION,
    description TEXT NOT NULL DEFAULT '',
    timeline JSONB NOT NULL DEFAULT '[]'::jsonb,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_work_orders_asset_id ON work_orders(asset_id);
CREATE INDEX idx_work_orders_status ON work_orders(status);
CREATE INDEX idx_work_orders_priority ON work_orders(priority);
CREATE INDEX idx_work_orders_created_at ON work_orders(created_at DESC);

-- +goose Down
DROP TABLE IF EXISTS work_orders;
