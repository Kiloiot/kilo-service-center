-- Audit script to identify cross-tenant duplicate EUIs before migration
-- **CRITICAL**: Run this manually BEFORE applying migration 072
-- **FAIL CONDITION**: If ANY row is returned, migration 072 will fail with EXCEPTION
-- **ACTION REQUIRED**: Coordinate with ops team to resolve duplicates before proceeding

-- Check for duplicate ep_eui across tenants
SELECT ep_eui, COUNT(DISTINCT tenant_id) as tenant_count,
       array_agg(DISTINCT tenant_id) as tenant_ids,
       array_agg(id) as endpoint_ids
FROM endpoints
GROUP BY ep_eui
HAVING COUNT(DISTINCT tenant_id) > 1
ORDER BY tenant_count DESC;

-- Check for duplicate bs_eui across tenants
SELECT bs_eui, COUNT(DISTINCT tenant_id) as tenant_count,
       array_agg(DISTINCT tenant_id) as tenant_ids,
       array_agg(id) as basestation_ids
FROM basestations
GROUP BY bs_eui
HAVING COUNT(DISTINCT tenant_id) > 1
ORDER BY tenant_count DESC;
