-- Rollback migration: Remove owner_tenant_id and roaming support (DEFER-005)
-- WARNING: This will permanently delete roaming event history

-- ============================================================================
-- 1. Drop roaming statistics view
-- ============================================================================

DROP VIEW IF EXISTS roaming_statistics;

-- ============================================================================
-- 2. Remove roaming configuration from tenants table
-- ============================================================================

ALTER TABLE tenants
DROP COLUMN IF EXISTS roaming_enabled,
DROP COLUMN IF EXISTS roaming_partners,
DROP COLUMN IF EXISTS roaming_policy;

-- ============================================================================
-- 3. Drop roaming_events table
-- ============================================================================

DROP TABLE IF EXISTS roaming_events;

-- ============================================================================
-- 4. Remove roaming tracking from basestation_sessions
-- ============================================================================

DROP INDEX IF EXISTS idx_basestation_sessions_roaming_endpoints;

ALTER TABLE basestation_sessions
DROP COLUMN IF EXISTS roaming_endpoint_count,
DROP COLUMN IF EXISTS roaming_endpoints;

-- ============================================================================
-- 5. Remove owner_tenant_id from messages
-- ============================================================================
-- Note: Migration 047 renamed mioty_messages → messages. This targets the correct table.

DROP INDEX IF EXISTS idx_messages_owner_tenant_id;

ALTER TABLE messages DROP COLUMN IF EXISTS owner_tenant_id;

-- ============================================================================
-- 6. Remove owner_tenant_id from endpoints
-- ============================================================================

DROP INDEX IF EXISTS idx_endpoints_roaming;
DROP INDEX IF EXISTS idx_endpoints_owner_tenant_id;

ALTER TABLE endpoints
DROP CONSTRAINT IF EXISTS fk_endpoints_owner_tenant;

ALTER TABLE endpoints
DROP COLUMN IF EXISTS owner_tenant_id;