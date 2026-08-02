-- +goose Up
-- Add organization_id to all tenant-scoped tables. Nullable so existing
-- single-tenant data can be backfilled into the default organization.
ALTER TABLE assets
    ADD COLUMN organization_id UUID REFERENCES organizations(id) ON DELETE SET NULL;
ALTER TABLE devices
    ADD COLUMN organization_id UUID REFERENCES organizations(id) ON DELETE SET NULL;
ALTER TABLE alerts
    ADD COLUMN organization_id UUID REFERENCES organizations(id) ON DELETE SET NULL;
ALTER TABLE work_orders
    ADD COLUMN organization_id UUID REFERENCES organizations(id) ON DELETE SET NULL;
ALTER TABLE actions
    ADD COLUMN organization_id UUID REFERENCES organizations(id) ON DELETE SET NULL;
ALTER TABLE digital_twins
    ADD COLUMN organization_id UUID REFERENCES organizations(id) ON DELETE SET NULL;

-- Backfill existing rows into the default organization (created in 015 seed
-- as a known UUID, inserted here idempotently).
INSERT INTO organizations (id, name, slug, created_at, updated_at)
VALUES ('00000000-0000-7000-8000-000000000001', 'Default Organization', 'default', NOW(), NOW())
ON CONFLICT (slug) DO NOTHING;

UPDATE assets SET organization_id = '00000000-0000-7000-8000-000000000001' WHERE organization_id IS NULL;
UPDATE devices SET organization_id = '00000000-0000-7000-8000-000000000001' WHERE organization_id IS NULL;
UPDATE alerts SET organization_id = '00000000-0000-7000-8000-000000000001' WHERE organization_id IS NULL;
UPDATE work_orders SET organization_id = '00000000-0000-7000-8000-000000000001' WHERE organization_id IS NULL;
UPDATE actions SET organization_id = '00000000-0000-7000-8000-000000000001' WHERE organization_id IS NULL;
UPDATE digital_twins SET organization_id = '00000000-0000-7000-8000-000000000001' WHERE organization_id IS NULL;

-- Add org scoping indexes
CREATE INDEX idx_assets_org_id ON assets(organization_id);
CREATE INDEX idx_devices_org_id ON devices(organization_id);
CREATE INDEX idx_alerts_org_id ON alerts(organization_id);
CREATE INDEX idx_work_orders_org_id ON work_orders(organization_id);
CREATE INDEX idx_actions_org_id ON actions(organization_id);

-- +goose Down
ALTER TABLE assets DROP COLUMN IF EXISTS organization_id;
ALTER TABLE devices DROP COLUMN IF EXISTS organization_id;
ALTER TABLE alerts DROP COLUMN IF EXISTS organization_id;
ALTER TABLE work_orders DROP COLUMN IF EXISTS organization_id;
ALTER TABLE actions DROP COLUMN IF EXISTS organization_id;
ALTER TABLE digital_twins DROP COLUMN IF EXISTS organization_id;
