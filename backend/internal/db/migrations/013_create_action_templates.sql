-- +goose Up
CREATE TABLE action_templates (
    id UUID PRIMARY KEY,
    name TEXT NOT NULL UNIQUE,
    action_type TEXT NOT NULL CHECK (action_type IN ('command', 'restart', 'config_change', 'notification')),
    payload JSONB NOT NULL DEFAULT '{}'::jsonb,
    auto_approve BOOLEAN NOT NULL DEFAULT FALSE,
    condition TEXT NOT NULL DEFAULT '',
    description TEXT NOT NULL DEFAULT '',
    active BOOLEAN NOT NULL DEFAULT TRUE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

INSERT INTO action_templates (id, name, action_type, payload, auto_approve, condition, description) VALUES
('11111111-1111-7111-8111-111111111111', 'restart_after_heartbeat_loss',
 'restart', '{"command":"restart"}', TRUE,
 'device.status_changed.to == "offline" AND device.offline_duration > 5m',
 'Automatically restart a device after prolonged heartbeat loss'),
('22222222-2222-7222-8222-222222222222', 'notify_high_temp',
 'notification', '{"channel":"dashboard","severity":"warning"}', TRUE,
 'telemetry.temperature > 90',
 'Notify dashboard when temperature is critically high'),
('33333333-3333-7333-8333-333333333333', 'config_reduce_load',
 'config_change', '{"config":{"load_target":80}}', FALSE,
 'telemetry.current > 180 AND telemetry.current > 180 for 10m',
 'Propose load reduction configuration change (requires operator approval)');

-- +goose Down
DROP TABLE IF EXISTS action_templates;
