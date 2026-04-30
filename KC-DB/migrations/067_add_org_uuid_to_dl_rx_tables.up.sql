-- Migration 067: Add Organization UUID to DL RX Status Tables
-- Purpose: Enable organization-level filtering and aggregation for multi-tenant audit
-- Related: BSSCI §5.15-5.16 telemetry, roaming architecture (see docs/architecture/roaming/ at repo root)

-- Add org_uuid to dl_rx_status table
ALTER TABLE dl_rx_status
    ADD COLUMN org_uuid UUID;

-- Add org_uuid to dl_rx_status_queries table
ALTER TABLE dl_rx_status_queries
    ADD COLUMN org_uuid UUID;

-- Create indexes for efficient organization-level queries
CREATE INDEX IF NOT EXISTS idx_dl_rx_status_org_tenant
    ON dl_rx_status(org_uuid, tenant_id, created_at DESC)
    WHERE org_uuid IS NOT NULL;

CREATE INDEX IF NOT EXISTS idx_dl_rx_status_queries_org_tenant
    ON dl_rx_status_queries(org_uuid, tenant_id, requested_at DESC)
    WHERE org_uuid IS NOT NULL;

-- Documentation
COMMENT ON COLUMN dl_rx_status.org_uuid IS
    'Organization UUID for multi-tenant isolation. Nullable for backward compatibility. Enables org-level analytics and roaming support per docs/architecture/roaming/ (repo root).';

COMMENT ON COLUMN dl_rx_status_queries.org_uuid IS
    'Organization UUID for multi-tenant isolation. Nullable for backward compatibility. Enables org-level query history filtering in KC-Web.';
