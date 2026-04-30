-- Migration: Remove deferred roaming columns
-- Purpose: Remove roaming_partners and roaming_policy columns that will be implemented in DEFER-006
-- Date: 2025-11-15
--
-- These columns were added in migration 075 but are not being used in the current
-- DEFER-005 implementation. They will be re-added when DEFER-006 (Global Roaming
-- with policy engine) is implemented.

-- IMPORTANT: Drop the view FIRST since it depends on roaming_partners column
DROP VIEW IF EXISTS roaming_statistics;

-- Drop the GIN index on roaming_partners
DROP INDEX IF EXISTS idx_tenants_roaming_partners;

-- Remove the unused columns
ALTER TABLE tenants
DROP COLUMN IF EXISTS roaming_partners,
DROP COLUMN IF EXISTS roaming_policy;

-- Recreate the roaming_statistics view without roaming_partners reference
CREATE OR REPLACE VIEW roaming_statistics AS
SELECT
    t.id as tenant_id,
    t.name as tenant_name,
    COUNT(DISTINCT e.id) FILTER (WHERE e.owner_tenant_id = t.id AND e.tenant_id != t.id) as owned_roaming_out,
    COUNT(DISTINCT e.id) FILTER (WHERE e.owner_tenant_id != t.id AND e.tenant_id = t.id) as visiting_roaming_in,
    COUNT(DISTINCT re.id) FILTER (WHERE re.owner_tenant_id = t.id AND re.created_at > NOW() - INTERVAL '24 hours') as roaming_events_24h,
    t.roaming_enabled
FROM tenants t
LEFT JOIN endpoints e ON t.id IN (e.owner_tenant_id, e.tenant_id)
LEFT JOIN roaming_events re ON t.id IN (re.owner_tenant_id, re.serving_tenant_id)
GROUP BY t.id, t.name, t.roaming_enabled;

-- Add comment explaining the deferral
COMMENT ON COLUMN tenants.roaming_enabled IS 'Whether this tenant participates in roaming. Partners and policies deferred to DEFER-006';