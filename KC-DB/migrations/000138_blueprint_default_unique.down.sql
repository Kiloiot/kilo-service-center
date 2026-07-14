DROP INDEX IF EXISTS uq_blueprints_default_per_model;
CREATE INDEX idx_blueprints_default ON blueprints(device_model_id) WHERE is_default = true;
