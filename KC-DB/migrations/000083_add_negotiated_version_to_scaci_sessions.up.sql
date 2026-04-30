-- Migration 000083: Add negotiated_version column to scaci_sessions
-- Per SCACI §§2.1-2.3: Version negotiation must be persisted for session resume validation.
-- Resume attempts with a different version than originally negotiated must be rejected.

ALTER TABLE scaci_sessions
ADD COLUMN negotiated_version VARCHAR(16);

-- Backfill existing sessions with default version (all existing sessions were v1.0.0)
UPDATE scaci_sessions SET negotiated_version = '1.0.0' WHERE negotiated_version IS NULL;

-- Make column NOT NULL after backfill to enforce integrity going forward
ALTER TABLE scaci_sessions ALTER COLUMN negotiated_version SET NOT NULL;

-- Set default for new sessions
ALTER TABLE scaci_sessions ALTER COLUMN negotiated_version SET DEFAULT '1.0.0';

COMMENT ON COLUMN scaci_sessions.negotiated_version IS 'Protocol version negotiated at connect per SCACI §2.1-2.3 (e.g., 1.0.0)';

-- Add partial index for efficient resume lookups by tenant + version
-- Pattern matches 000082 (TLS metadata index)
CREATE INDEX idx_scaci_sessions_tenant_version
ON scaci_sessions (tenant_id, negotiated_version)
WHERE negotiated_version IS NOT NULL;
