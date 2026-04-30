-- Rollback migration 073: Remove organization UUID from messages table

DROP INDEX IF EXISTS idx_messages_ep_org;
DROP INDEX IF EXISTS idx_messages_org_tenant_time;
ALTER TABLE messages DROP COLUMN IF EXISTS org_uuid;
