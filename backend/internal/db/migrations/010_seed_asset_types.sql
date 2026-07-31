-- +goose Up
INSERT INTO asset_types (type, display_name, metrics, thresholds, health_weights) VALUES
('transformer', 'Transformer',
 '[{"name":"temperature","unit":"°C"},{"name":"current","unit":"A"},{"name":"voltage","unit":"V"},{"name":"humidity","unit":"%"}]',
 '{"temperature":{"warning":75,"critical":90},"current":{"warning":120,"critical":180},"voltage":{"min":10000},"humidity":{"max":80}}',
 '{"temperature":0.4,"current":0.3,"voltage":0.15,"humidity":0.15}'),
('pump', 'Pump',
 '[{"name":"temperature","unit":"°C"},{"name":"flow_rate","unit":"m³/h"},{"name":"pressure","unit":"bar"},{"name":"vibration","unit":"mm/s"}]',
 '{"temperature":{"warning":80,"critical":95},"flow_rate":{"min":50},"pressure":{"warning":8,"critical":10},"vibration":{"warning":4.5,"critical":7.1}}',
 '{"temperature":0.3,"flow_rate":0.25,"pressure":0.25,"vibration":0.2}'),
('motor', 'Motor',
 '[{"name":"temperature","unit":"°C"},{"name":"rpm","unit":"rpm"},{"name":"current","unit":"A"},{"name":"vibration","unit":"mm/s"}]',
 '{"temperature":{"warning":85,"critical":100},"rpm":{"min":1400},"current":{"warning":110,"critical":150},"vibration":{"warning":3.5,"critical":5.5}}',
 '{"temperature":0.3,"rpm":0.2,"current":0.3,"vibration":0.2}'),
('generator', 'Generator',
 '[{"name":"temperature","unit":"°C"},{"name":"output_power","unit":"kW"},{"name":"frequency","unit":"Hz"},{"name":"voltage","unit":"V"}]',
 '{"temperature":{"warning":80,"critical":95},"output_power":{"min":100},"frequency":{"warning":49.5,"critical":49.0},"voltage":{"min":11000}}',
 '{"temperature":0.35,"output_power":0.3,"frequency":0.2,"voltage":0.15}'),
('hvac', 'HVAC System',
 '[{"name":"temperature","unit":"°C"},{"name":"supply_air_temp","unit":"°C"},{"name":"power","unit":"kW"},{"name":"humidity","unit":"%"}]',
 '{"temperature":{"warning":35,"critical":45},"supply_air_temp":{"warning":18,"critical":12},"power":{"warning":80,"critical":95},"humidity":{"max":70}}',
 '{"temperature":0.3,"supply_air_temp":0.3,"power":0.2,"humidity":0.2}');

ALTER TABLE assets
    ADD CONSTRAINT fk_assets_type FOREIGN KEY (type) REFERENCES asset_types(type);

-- +goose Down
ALTER TABLE assets DROP CONSTRAINT IF EXISTS fk_assets_type;
DELETE FROM asset_types WHERE type IN ('transformer','pump','motor','generator','hvac');
