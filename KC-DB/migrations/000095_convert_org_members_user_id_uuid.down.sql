-- Revert user_id from UUID to VARCHAR
DROP INDEX IF EXISTS idx_org_members_user;
ALTER TABLE organization_members DROP CONSTRAINT IF EXISTS fk_org_members_user;
ALTER TABLE organization_members ALTER COLUMN user_id TYPE VARCHAR(255) USING user_id::text;
CREATE INDEX idx_org_members_user ON organization_members(user_id, status);
