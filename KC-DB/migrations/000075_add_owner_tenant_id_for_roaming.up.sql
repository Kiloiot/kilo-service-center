-- Migration: Add owner_tenant_id for roaming support (DEFER-005)
-- Purpose: Enable multi-tenant roaming by tracking endpoint ownership
-- Date: 2025-11-15
--
-- Roaming allows endpoints to connect through any base station regardless
-- of tenant ownership. The owner_tenant_id tracks the original owner while
-- tenant_id represents the current serving tenant.

-- ============================================================================
-- 1. Add owner_tenant_id to endpoints table
-- ============================================================================

-- Add owner_tenant_id column to track the original owner of the endpoint
-- This enables roaming scenarios where an endpoint owned by one tenant
-- can be served by base stations belonging to other tenants
ALTER TABLE endpoints
ADD COLUMN owner_tenant_id BIGINT;

-- Set initial values: owner_tenant_id defaults to current tenant_id
-- This preserves existing behavior for non-roaming endpoints
UPDATE endpoints
SET owner_tenant_id = tenant_id
WHERE owner_tenant_id IS NULL;

-- Make owner_tenant_id NOT NULL after backfilling
ALTER TABLE endpoints
ALTER COLUMN owner_tenant_id SET NOT NULL;

-- Add foreign key constraint to tenants table
ALTER TABLE endpoints
ADD CONSTRAINT fk_endpoints_owner_tenant
FOREIGN KEY (owner_tenant_id) REFERENCES tenants(id);

-- Create index for efficient owner lookups
CREATE INDEX idx_endpoints_owner_tenant_id ON endpoints(owner_tenant_id);

-- Create composite index for roaming queries (find endpoints by owner in serving tenant)
CREATE INDEX idx_endpoints_roaming ON endpoints(owner_tenant_id, tenant_id)
WHERE owner_tenant_id != tenant_id;

-- ============================================================================
-- 2. Add owner_tenant_id to messages for historical tracking
-- ============================================================================
-- Note: Migration 047 renamed mioty_messages → messages. This targets the correct table.

-- Add owner_tenant_id to messages to preserve ownership history
ALTER TABLE messages ADD COLUMN IF NOT EXISTS owner_tenant_id BIGINT;

-- Backfill owner_tenant_id from endpoints with BYTEA→BIGINT conversion
-- Note: messages.ep_eui is BIGINT, endpoints.ep_eui is BYTEA (8 bytes)
UPDATE messages m
SET owner_tenant_id = e.owner_tenant_id
FROM endpoints e
WHERE m.ep_eui = (('x' || encode(e.ep_eui, 'hex'))::bit(64)::bigint)
  AND m.tenant_id = e.tenant_id
  AND m.owner_tenant_id IS NULL;

-- For messages without matching endpoints, use serving tenant as owner
UPDATE messages
SET owner_tenant_id = tenant_id
WHERE owner_tenant_id IS NULL;

-- Add index for owner-based message queries
CREATE INDEX IF NOT EXISTS idx_messages_owner_tenant_id ON messages(owner_tenant_id);

-- ============================================================================
-- 3. Add roaming status tracking to basestation_sessions
-- ============================================================================

-- Add fields to track roaming endpoints in base station sessions
ALTER TABLE basestation_sessions
ADD COLUMN roaming_endpoint_count INTEGER DEFAULT 0,
ADD COLUMN roaming_endpoints JSONB DEFAULT '[]'::jsonb;

-- Add index for roaming endpoint queries
CREATE INDEX idx_basestation_sessions_roaming_endpoints
ON basestation_sessions USING GIN(roaming_endpoints);

-- ============================================================================
-- 4. Create roaming_events table for audit trail
-- ============================================================================

CREATE TABLE roaming_events (
    id BIGSERIAL PRIMARY KEY,
    event_type VARCHAR(50) NOT NULL, -- 'attach', 'detach', 'handover'
    ep_eui BYTEA NOT NULL,
    owner_tenant_id BIGINT NOT NULL,
    serving_tenant_id BIGINT NOT NULL,
    from_bs_eui BYTEA,
    to_bs_eui BYTEA,
    reason TEXT,
    metadata JSONB,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    CONSTRAINT fk_roaming_events_owner_tenant
        FOREIGN KEY (owner_tenant_id) REFERENCES tenants(id),
    CONSTRAINT fk_roaming_events_serving_tenant
        FOREIGN KEY (serving_tenant_id) REFERENCES tenants(id)
);

-- Create indexes for roaming event queries
CREATE INDEX idx_roaming_events_ep_eui ON roaming_events(ep_eui);
CREATE INDEX idx_roaming_events_owner_tenant ON roaming_events(owner_tenant_id);
CREATE INDEX idx_roaming_events_serving_tenant ON roaming_events(serving_tenant_id);
CREATE INDEX idx_roaming_events_created_at ON roaming_events(created_at);
CREATE INDEX idx_roaming_events_type ON roaming_events(event_type);

-- ============================================================================
-- 5. Add roaming configuration to tenants table
-- ============================================================================

ALTER TABLE tenants
ADD COLUMN roaming_enabled BOOLEAN DEFAULT false,
ADD COLUMN roaming_partners JSONB DEFAULT '[]'::jsonb,
ADD COLUMN roaming_policy JSONB DEFAULT '{}'::jsonb;

-- Create index for roaming partner lookups
CREATE INDEX idx_tenants_roaming_partners
ON tenants USING GIN(roaming_partners);

-- ============================================================================
-- 6. Create view for roaming statistics
-- ============================================================================

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

-- ============================================================================
-- Comments for documentation
-- ============================================================================

COMMENT ON COLUMN endpoints.owner_tenant_id IS 'Original owner of the endpoint (for roaming support)';
COMMENT ON COLUMN endpoints.tenant_id IS 'Current serving tenant (may differ from owner during roaming)';
COMMENT ON COLUMN messages.owner_tenant_id IS 'Owner tenant ID for roaming support (may differ from tenant_id during roaming)';
COMMENT ON COLUMN basestation_sessions.roaming_endpoint_count IS 'Number of roaming endpoints in this session';
COMMENT ON COLUMN basestation_sessions.roaming_endpoints IS 'Array of roaming endpoint EUIs [{eui, owner_tenant_id}]';
COMMENT ON TABLE roaming_events IS 'Audit trail for endpoint roaming activities';
COMMENT ON COLUMN tenants.roaming_enabled IS 'Whether this tenant participates in roaming';
COMMENT ON COLUMN tenants.roaming_partners IS 'Array of partner tenant IDs allowed for roaming';
COMMENT ON COLUMN tenants.roaming_policy IS 'JSON policy object defining roaming rules and limits';
COMMENT ON VIEW roaming_statistics IS 'Aggregated view of roaming metrics per tenant';