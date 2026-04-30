-- Migration 071 Rollback: Remove Organization UUID from messages Table

-- Drop indexes first
DROP INDEX IF EXISTS idx_messages_tenant_org_type;
DROP INDEX IF EXISTS idx_messages_org_propagate;

-- Drop org_uuid column
ALTER TABLE messages DROP COLUMN IF EXISTS org_uuid;
