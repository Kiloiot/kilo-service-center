BEGIN;

-- Drop tenant-scoped performance indexes
DROP INDEX IF EXISTS idx_endpoints_tenant_ep_eui;
DROP INDEX IF EXISTS idx_basestations_tenant_bs_eui;

-- Drop global unique constraints
ALTER TABLE endpoints DROP CONSTRAINT IF EXISTS unique_ep_eui;
ALTER TABLE basestations DROP CONSTRAINT IF EXISTS unique_bs_eui;

-- Restore per-tenant composite UNIQUE constraints
ALTER TABLE endpoints ADD CONSTRAINT unique_endpoint_per_tenant UNIQUE(ep_eui, tenant_id);
ALTER TABLE basestations ADD CONSTRAINT unique_basestation_per_tenant UNIQUE(bs_eui, tenant_id);

COMMIT;
