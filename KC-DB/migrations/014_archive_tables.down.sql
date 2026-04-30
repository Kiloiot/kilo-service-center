-- Drop archive tables and related objects

-- Step 0: Drop ALL indexes on archive tables FIRST (17 total)
-- Documented in phase3-014-index-catalog.txt

-- messages_archive indexes (4):
DROP INDEX IF EXISTS idx_messages_archive_ep_eui;
DROP INDEX IF EXISTS idx_messages_archive_tenant_id;
DROP INDEX IF EXISTS idx_messages_archive_received_at;
DROP INDEX IF EXISTS idx_messages_archive_archived_at;

-- basestation_receptions_archive indexes (4):
DROP INDEX IF EXISTS idx_basestation_receptions_archive_message_id;
DROP INDEX IF EXISTS idx_basestation_receptions_archive_basestation_id;
DROP INDEX IF EXISTS idx_basestation_receptions_archive_created_at;
DROP INDEX IF EXISTS idx_basestation_receptions_archive_archived_at;

-- endpoint_sessions_archive indexes (5):
DROP INDEX IF EXISTS idx_endpoint_sessions_archive_endpoint_id;
DROP INDEX IF EXISTS idx_endpoint_sessions_archive_tenant_id;
DROP INDEX IF EXISTS idx_endpoint_sessions_archive_status;
DROP INDEX IF EXISTS idx_endpoint_sessions_archive_ended_at;
DROP INDEX IF EXISTS idx_endpoint_sessions_archive_archived_at;

-- endpoint_keys_archive indexes (4):
DROP INDEX IF EXISTS idx_endpoint_keys_archive_endpoint_id;
DROP INDEX IF EXISTS idx_endpoint_keys_archive_tenant_id;
DROP INDEX IF EXISTS idx_endpoint_keys_archive_key_type;
DROP INDEX IF EXISTS idx_endpoint_keys_archive_archived_at;

-- Step 1: Drop ALL triggers created by this migration (EXACT names from up script, case-sensitive)
DROP TRIGGER IF EXISTS set_messages_archive_timestamp ON messages_archive;
DROP TRIGGER IF EXISTS set_basestation_receptions_archive_timestamp ON basestation_receptions_archive;
DROP TRIGGER IF EXISTS set_endpoint_sessions_archive_timestamp ON endpoint_sessions_archive;
DROP TRIGGER IF EXISTS set_endpoint_keys_archive_timestamp ON endpoint_keys_archive;

-- Step 2: Now safe to drop function (all triggers removed)
DROP FUNCTION IF EXISTS set_archived_at();

-- Step 3: Drop archive tables (using current table names post-migration-060)
DROP TABLE IF EXISTS endpoint_keys_archive;
DROP TABLE IF EXISTS endpoint_sessions_archive;
DROP TABLE IF EXISTS basestation_receptions_archive;
-- Note: messages_archive might have existed before this migration, so we don't drop it
