-- Convert organization_members.user_id from VARCHAR to UUID
-- PREREQUISITE: Run audit query to verify all user_id values are valid UUIDs:
--   SELECT user_id FROM organization_members
--   WHERE user_id IS NOT NULL AND user_id !~* '^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$';
-- Migration will FAIL if any invalid UUID values exist

-- Drop existing index
DROP INDEX IF EXISTS idx_org_members_user;

-- Alter column type (fails if any non-UUID values)
ALTER TABLE organization_members
    ALTER COLUMN user_id TYPE UUID USING user_id::uuid;

-- Add foreign key constraint to users table
ALTER TABLE organization_members
    ADD CONSTRAINT fk_org_members_user
    FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE;

-- Recreate index with UUID type
CREATE INDEX idx_org_members_user ON organization_members(user_id, status);
