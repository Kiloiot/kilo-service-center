-- Migration 030: Add organization_id to session tables
-- Strategy: Nullable DEFAULT NULL - existing sessions backfill on reconnect

-- SCACI sessions
ALTER TABLE scaci_sessions
ADD COLUMN organization_id UUID DEFAULT NULL
  REFERENCES organizations(org_id) ON DELETE SET NULL;

CREATE INDEX idx_scaci_sessions_org
  ON scaci_sessions(organization_id)
  WHERE organization_id IS NOT NULL;

COMMENT ON COLUMN scaci_sessions.organization_id IS
'Organization UUID from TLS cert CN. NULL for existing sessions or community mode.';

-- BSSCI sessions (basestation_sessions from migration 028)
ALTER TABLE basestation_sessions
ADD COLUMN organization_id UUID DEFAULT NULL
  REFERENCES organizations(org_id) ON DELETE SET NULL;

CREATE INDEX idx_basestation_sessions_org
  ON basestation_sessions(organization_id)
  WHERE organization_id IS NOT NULL;

COMMENT ON COLUMN basestation_sessions.organization_id IS
'Organization UUID for base station owner. Propagates to messages. NULL allowed.';

-- Composite index for org-scoped session queries
DROP INDEX IF EXISTS idx_basestation_sessions_tenant;
CREATE INDEX idx_basestation_sessions_tenant_org
  ON basestation_sessions(tenant_id, organization_id, status)
  WHERE status IN ('active', 'resumed');
