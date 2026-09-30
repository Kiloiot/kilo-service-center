-- Reverts 000185: the streams of the previous release read by reception and
-- occurrence time and never read stored_at or the storage-order indexes.

DROP TRIGGER IF EXISTS system_events_restamp_moved ON system_events;
DROP FUNCTION IF EXISTS restamp_moved_event();
DROP INDEX IF EXISTS idx_system_events_tenant_stored;
ALTER TABLE system_events DROP COLUMN IF EXISTS stored_at;
DROP INDEX IF EXISTS idx_messages_tenant_stored;
