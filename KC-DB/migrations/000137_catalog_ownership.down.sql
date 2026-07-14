-- Fails if System rows or duplicate (tenant, type_eui) exist — remove first.

-- ── blueprints ──
ALTER TABLE blueprints ADD CONSTRAINT uq_blueprints_tenant_type_eui UNIQUE(tenant_id, type_eui);
ALTER TABLE blueprints DROP CONSTRAINT IF EXISTS chk_blueprints_ownership;
ALTER TABLE blueprints DROP COLUMN IF EXISTS is_system;
ALTER TABLE blueprints ALTER COLUMN tenant_id SET NOT NULL;

-- ── device_models ──
ALTER TABLE device_models DROP CONSTRAINT IF EXISTS chk_device_models_ownership;
ALTER TABLE device_models DROP COLUMN IF EXISTS is_system;
ALTER TABLE device_models ALTER COLUMN tenant_id SET NOT NULL;

-- ── manufacturers ──
DROP INDEX IF EXISTS uq_manufacturers_system_name_ci;
DROP INDEX IF EXISTS uq_manufacturers_tenant_name_ci;
CREATE UNIQUE INDEX uq_manufacturers_tenant_name_ci ON manufacturers(tenant_id, LOWER(name));
ALTER TABLE manufacturers DROP CONSTRAINT IF EXISTS chk_manufacturers_ownership;
ALTER TABLE manufacturers DROP COLUMN IF EXISTS is_system;
ALTER TABLE manufacturers ALTER COLUMN tenant_id SET NOT NULL;
