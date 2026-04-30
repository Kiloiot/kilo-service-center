-- Expression indexes for case-insensitive ORDER BY LOWER(name) on device_models.
-- The manufacturers table already has uq_manufacturers_tenant_name_ci from migration 000107.
CREATE INDEX idx_device_models_tenant_name_ci ON device_models(tenant_id, LOWER(name), name, id);
CREATE INDEX idx_device_models_mfr_name_ci ON device_models(tenant_id, manufacturer_id, LOWER(name), name, id);
