-- Drop FK constraints that reference tenants table (pre-053 constraints)
-- These must be dropped before the tenants table can be dropped
-- Order matches the 13 pre-053 FK constraints documented in phase3-fk-research.txt

-- 1. endpoints_tenant_id_fkey
ALTER TABLE endpoints DROP CONSTRAINT IF EXISTS endpoints_tenant_id_fkey;

-- 2. basestations_tenant_id_fkey
ALTER TABLE basestations DROP CONSTRAINT IF EXISTS basestations_tenant_id_fkey;

-- 3. downlink_queue_tenant_id_fkey
ALTER TABLE downlink_queue DROP CONSTRAINT IF EXISTS downlink_queue_tenant_id_fkey;

-- 4. roaming_agreements_tenant_id_fkey
ALTER TABLE roaming_agreements DROP CONSTRAINT IF EXISTS roaming_agreements_tenant_id_fkey;

-- 5. basestation_receptions_tenant_id_fkey
ALTER TABLE basestation_receptions DROP CONSTRAINT IF EXISTS basestation_receptions_tenant_id_fkey;

-- 6. endpoint_sessions_tenant_id_fkey1
ALTER TABLE endpoint_sessions DROP CONSTRAINT IF EXISTS endpoint_sessions_tenant_id_fkey1;

-- 7. endpoint_keys_tenant_id_fkey1
ALTER TABLE endpoint_keys DROP CONSTRAINT IF EXISTS endpoint_keys_tenant_id_fkey1;

-- 8. basestation_sessions_tenant_id_fkey1
ALTER TABLE basestation_sessions DROP CONSTRAINT IF EXISTS basestation_sessions_tenant_id_fkey1;

-- 9. system_events_tenant_id_fkey
ALTER TABLE system_events DROP CONSTRAINT IF EXISTS system_events_tenant_id_fkey;

-- 10. mioty_subpackets_tenant_id_fkey
ALTER TABLE mioty_subpackets DROP CONSTRAINT IF EXISTS mioty_subpackets_tenant_id_fkey;

-- 11. mioty_basestation_status_tenant_id_fkey
ALTER TABLE mioty_basestation_status DROP CONSTRAINT IF EXISTS mioty_basestation_status_tenant_id_fkey;

-- 12. mioty_message_deduplication_tenant_id_fkey
ALTER TABLE mioty_message_deduplication DROP CONSTRAINT IF EXISTS mioty_message_deduplication_tenant_id_fkey;

-- 13. dl_rx_status_tenant_id_fkey
ALTER TABLE dl_rx_status DROP CONSTRAINT IF EXISTS dl_rx_status_tenant_id_fkey;

-- Drop unique constraint on name
ALTER TABLE tenants DROP CONSTRAINT IF EXISTS tenants_name_key;

-- Drop indexes
DROP INDEX IF EXISTS idx_tenants_status;
DROP INDEX IF EXISTS idx_tenants_name;

-- Drop tenants table (now safe - all FK constraints removed)
DROP TABLE IF EXISTS tenants;