-- Migration 072: Enforce global EUI uniqueness + retain tenant-scoped indexes
-- **CRITICAL PREREQUISITE**: Run KC-DB/migrations/scripts/audit_cross_tenant_duplicates.sql FIRST
-- **FAIL-SAFE**: Migration will abort with EXCEPTION if ANY cross-tenant duplicates exist
-- **RELEASE STEP**: This audit + migration sequence must be documented in release notes

BEGIN;

-- Pre-flight check: FAIL HARD if cross-tenant duplicates exist
DO $$
DECLARE
    ep_duplicate_count INT;
    bs_duplicate_count INT;
BEGIN
    SELECT COUNT(*) INTO ep_duplicate_count
    FROM (
        SELECT ep_eui
        FROM endpoints
        GROUP BY ep_eui
        HAVING COUNT(DISTINCT tenant_id) > 1
    ) duplicates;

    SELECT COUNT(*) INTO bs_duplicate_count
    FROM (
        SELECT bs_eui
        FROM basestations
        GROUP BY bs_eui
        HAVING COUNT(DISTINCT tenant_id) > 1
    ) duplicates;

    IF ep_duplicate_count > 0 OR bs_duplicate_count > 0 THEN
        RAISE EXCEPTION 'Cross-tenant EUI duplicates detected. Endpoints: %, Basestations: %. Run audit script (migrations/scripts/audit_cross_tenant_duplicates.sql) and resolve before proceeding.',
            ep_duplicate_count, bs_duplicate_count;
    END IF;
END $$;

-- Drop existing per-tenant composite UNIQUE constraints
ALTER TABLE endpoints DROP CONSTRAINT IF EXISTS unique_endpoint_per_tenant;
ALTER TABLE basestations DROP CONSTRAINT IF EXISTS unique_basestation_per_tenant;

-- Add global UNIQUE constraints (enforce hardware reality: EUIs are globally unique)
ALTER TABLE endpoints ADD CONSTRAINT unique_ep_eui UNIQUE(ep_eui);
ALTER TABLE basestations ADD CONSTRAINT unique_bs_eui UNIQUE(bs_eui);

-- **PERFORMANCE**: Retain tenant-scoped non-unique indexes for fast tenant-filtered queries
-- Note: These indexes support "WHERE tenant_id = X" lookups without full table scans
CREATE INDEX IF NOT EXISTS idx_endpoints_tenant_ep_eui ON endpoints(tenant_id, ep_eui);
CREATE INDEX IF NOT EXISTS idx_basestations_tenant_bs_eui ON basestations(tenant_id, bs_eui);

COMMIT;
