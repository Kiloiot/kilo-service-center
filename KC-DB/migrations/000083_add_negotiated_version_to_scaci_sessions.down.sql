-- Migration 000083 down: Remove negotiated_version column from scaci_sessions

-- Drop partial index first
DROP INDEX IF EXISTS idx_scaci_sessions_tenant_version;

ALTER TABLE scaci_sessions DROP COLUMN IF EXISTS negotiated_version;
