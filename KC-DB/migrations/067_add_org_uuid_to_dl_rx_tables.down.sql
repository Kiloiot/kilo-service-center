-- Migration 067 Rollback: Remove Organization UUID from DL RX Status Tables

-- Drop indexes
DROP INDEX IF EXISTS idx_dl_rx_status_org_tenant;
DROP INDEX IF EXISTS idx_dl_rx_status_queries_org_tenant;

-- Remove org_uuid columns
ALTER TABLE dl_rx_status
    DROP COLUMN IF EXISTS org_uuid;

ALTER TABLE dl_rx_status_queries
    DROP COLUMN IF EXISTS org_uuid;
