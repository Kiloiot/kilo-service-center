-- Differentiate EUI fields: rename 'eui' to 'bs_eui' in basestations and 'ep_eui' in endpoints
-- This provides clearer field naming and better schema structure (snake_case per PostgreSQL conventions)

-- Update basestations table: rename eui to bs_eui
ALTER TABLE basestations RENAME COLUMN eui TO bs_eui;

-- Update endpoints table: rename eui to ep_eui
ALTER TABLE endpoints RENAME COLUMN eui TO ep_eui;

-- Note: messages table already uses ep_eui and bs_eui (snake_case) - keep them as-is

-- Update constraint names to reflect new field names
ALTER TABLE basestations RENAME CONSTRAINT unique_basestation_per_tenant TO unique_basestation_per_tenant_bseui;
ALTER TABLE basestations DROP CONSTRAINT IF EXISTS unique_basestation_per_tenant;
ALTER TABLE basestations ADD CONSTRAINT unique_basestation_per_tenant UNIQUE(bs_eui, tenant_id);
-- Drop the renamed constraint to avoid duplicates
ALTER TABLE basestations DROP CONSTRAINT IF EXISTS unique_basestation_per_tenant_bseui;

ALTER TABLE endpoints RENAME CONSTRAINT unique_endpoint_per_tenant TO unique_endpoint_per_tenant_epeui;
ALTER TABLE endpoints DROP CONSTRAINT IF EXISTS unique_endpoint_per_tenant;
ALTER TABLE endpoints ADD CONSTRAINT unique_endpoint_per_tenant UNIQUE(ep_eui, tenant_id);
-- Drop the renamed constraint to avoid duplicates
ALTER TABLE endpoints DROP CONSTRAINT IF EXISTS unique_endpoint_per_tenant_epeui;

-- Update any indexes that reference the old field names
-- Note: PostgreSQL automatically updates indexes when columns are renamed, but we document this for clarity

-- Add comments to document the new field naming convention (snake_case per PostgreSQL standards)
COMMENT ON COLUMN basestations.bs_eui IS 'Base Station EUI (8-byte Extended Unique Identifier) - identifies MIOTY Base Station';
COMMENT ON COLUMN endpoints.ep_eui IS 'End Point EUI (8-byte Extended Unique Identifier) - identifies MIOTY End Point';
COMMENT ON COLUMN messages.bs_eui IS 'Base Station EUI (8-byte Extended Unique Identifier) - receiving Base Station';
COMMENT ON COLUMN messages.ep_eui IS 'End Point EUI (8-byte Extended Unique Identifier) - originating End Point';
