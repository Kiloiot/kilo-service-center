-- Migration 000097: Remove external_id column from organizations table

-- Drop the unique index first
DROP INDEX IF EXISTS idx_organizations_external_id;

-- Remove the external_id column
ALTER TABLE organizations
    DROP COLUMN IF EXISTS external_id;
