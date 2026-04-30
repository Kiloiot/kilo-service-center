-- Migration 029 Rollback: Remove Organization/Tenant Bridge

-- Drop organization members first (foreign key dependency)
DROP TABLE IF EXISTS organization_members CASCADE;

-- Drop organizations table
DROP TABLE IF EXISTS organizations CASCADE;

-- Note: This rollback is safe as long as no production code depends on these tables yet
-- rolling back would break organization resolution
