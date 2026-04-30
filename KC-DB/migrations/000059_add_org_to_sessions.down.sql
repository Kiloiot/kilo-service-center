-- Rollback migration 030

-- Drop BSSCI indexes and column
DROP INDEX IF EXISTS idx_basestation_sessions_tenant_org;
DROP INDEX IF EXISTS idx_basestation_sessions_org;
ALTER TABLE basestation_sessions DROP COLUMN IF EXISTS organization_id;

-- Recreate original tenant index
CREATE INDEX IF NOT EXISTS idx_basestation_sessions_tenant
  ON basestation_sessions(tenant_id, status);

-- Drop SCACI index and column
DROP INDEX IF EXISTS idx_scaci_sessions_org;
ALTER TABLE scaci_sessions DROP COLUMN IF EXISTS organization_id;
