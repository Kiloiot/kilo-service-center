-- Rollback migration 093: Remove positive tenant_id constraints

ALTER TABLE messages DROP CONSTRAINT IF EXISTS valid_tenant_id;
ALTER TABLE messages DROP CONSTRAINT IF EXISTS valid_owner_tenant_id;
