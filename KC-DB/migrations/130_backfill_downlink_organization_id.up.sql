-- Backfill organization_id on downlink tables using deterministic default org per tenant
-- Matches organization_repository.go GetOrgByTenantID semantics (ORDER BY created_at ASC LIMIT 1)

WITH default_orgs AS (
    SELECT DISTINCT ON (tenant_id) tenant_id, org_id
    FROM organizations
    ORDER BY tenant_id, created_at ASC, org_id ASC
)
UPDATE downlink_queue dq
SET organization_id = d.org_id
FROM default_orgs d
WHERE d.tenant_id = dq.tenant_id AND dq.organization_id IS NULL;

WITH default_orgs AS (
    SELECT DISTINCT ON (tenant_id) tenant_id, org_id
    FROM organizations
    ORDER BY tenant_id, created_at ASC, org_id ASC
)
UPDATE dl_rx_status drs
SET org_uuid = d.org_id
FROM default_orgs d
WHERE d.tenant_id = drs.tenant_id AND drs.org_uuid IS NULL;

WITH default_orgs AS (
    SELECT DISTINCT ON (tenant_id) tenant_id, org_id
    FROM organizations
    ORDER BY tenant_id, created_at ASC, org_id ASC
)
UPDATE dl_rx_status_queries drsq
SET org_uuid = d.org_id
FROM default_orgs d
WHERE d.tenant_id = drsq.tenant_id AND drsq.org_uuid IS NULL;
