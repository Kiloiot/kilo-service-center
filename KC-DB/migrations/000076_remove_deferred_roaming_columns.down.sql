-- Rollback: Re-add deferred roaming columns
-- This restores the columns that were removed because they're deferred to DEFER-006

-- Re-add the columns
ALTER TABLE tenants
ADD COLUMN roaming_partners JSONB DEFAULT '[]'::jsonb,
ADD COLUMN roaming_policy JSONB DEFAULT '{}'::jsonb;

-- Recreate the index
CREATE INDEX idx_tenants_roaming_partners
ON tenants USING GIN(roaming_partners);

-- Restore the original view with partner_count
CREATE OR REPLACE VIEW roaming_statistics AS
SELECT
    t.id as tenant_id,
    t.name as tenant_name,
    COUNT(DISTINCT e.id) FILTER (WHERE e.owner_tenant_id = t.id AND e.tenant_id != t.id) as owned_roaming_out,
    COUNT(DISTINCT e.id) FILTER (WHERE e.owner_tenant_id != t.id AND e.tenant_id = t.id) as visiting_roaming_in,
    COUNT(DISTINCT re.id) FILTER (WHERE re.owner_tenant_id = t.id AND re.created_at > NOW() - INTERVAL '24 hours') as roaming_events_24h,
    t.roaming_enabled,
    jsonb_array_length(t.roaming_partners) as partner_count
FROM tenants t
LEFT JOIN endpoints e ON t.id IN (e.owner_tenant_id, e.tenant_id)
LEFT JOIN roaming_events re ON t.id IN (re.owner_tenant_id, re.serving_tenant_id)
GROUP BY t.id, t.name, t.roaming_enabled, t.roaming_partners;

-- Restore original comments
COMMENT ON COLUMN tenants.roaming_enabled IS 'Whether this tenant participates in roaming';
COMMENT ON COLUMN tenants.roaming_partners IS 'Array of partner tenant IDs allowed for roaming';
COMMENT ON COLUMN tenants.roaming_policy IS 'JSON policy object defining roaming rules and limits';