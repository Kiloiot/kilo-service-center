-- System rows (tenant_id NULL) escape the tenant ON DELETE CASCADE.

-- ── manufacturers ──
ALTER TABLE manufacturers ALTER COLUMN tenant_id DROP NOT NULL;
ALTER TABLE manufacturers ADD COLUMN IF NOT EXISTS is_system BOOLEAN NOT NULL DEFAULT false;
ALTER TABLE manufacturers ADD CONSTRAINT chk_manufacturers_ownership
    CHECK ((is_system AND tenant_id IS NULL) OR (NOT is_system AND tenant_id IS NOT NULL));
-- Separate tenant and system name uniqueness; tenant partial matches the old plain index (no dedup needed).
DROP INDEX IF EXISTS uq_manufacturers_tenant_name_ci;
CREATE UNIQUE INDEX uq_manufacturers_tenant_name_ci
    ON manufacturers(tenant_id, LOWER(name)) WHERE tenant_id IS NOT NULL;
CREATE UNIQUE INDEX uq_manufacturers_system_name_ci
    ON manufacturers(LOWER(name)) WHERE is_system;

-- ── device_models ── (uniqueness unchanged: parent manufacturer_id segregates tenant from system)
ALTER TABLE device_models ALTER COLUMN tenant_id DROP NOT NULL;
ALTER TABLE device_models ADD COLUMN IF NOT EXISTS is_system BOOLEAN NOT NULL DEFAULT false;
ALTER TABLE device_models ADD CONSTRAINT chk_device_models_ownership
    CHECK ((is_system AND tenant_id IS NULL) OR (NOT is_system AND tenant_id IS NOT NULL));

-- ── blueprints ──
ALTER TABLE blueprints ALTER COLUMN tenant_id DROP NOT NULL;
ALTER TABLE blueprints ADD COLUMN IF NOT EXISTS is_system BOOLEAN NOT NULL DEFAULT false;
ALTER TABLE blueprints ADD CONSTRAINT chk_blueprints_ownership
    CHECK ((is_system AND tenant_id IS NULL) OR (NOT is_system AND tenant_id IS NOT NULL));
-- Drop tenant type_eui uniqueness: blocks multiple versions sharing a type_eui.
ALTER TABLE blueprints DROP CONSTRAINT IF EXISTS uq_blueprints_tenant_type_eui;

COMMENT ON COLUMN manufacturers.is_system IS 'System row (tenant_id NULL) vs tenant Custom row';
COMMENT ON COLUMN device_models.is_system IS 'System row (tenant_id NULL) vs tenant Custom row';
COMMENT ON COLUMN blueprints.is_system IS 'System row (tenant_id NULL) vs tenant Custom row';
