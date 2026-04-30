-- Migration 000107: Simplify manufacturers table
-- Remove code, description, logo_url columns; add case-insensitive name uniqueness
--
-- PREREQUISITE CHECK (run before applying migration):
--   SELECT tenant_id, LOWER(name) as name_lower, COUNT(*)
--   FROM manufacturers
--   GROUP BY tenant_id, LOWER(name)
--   HAVING COUNT(*) > 1;
--
-- If duplicates exist, resolve them manually before running this migration.
-- Options: rename duplicates, merge records, or delete obsolete entries.
-- Failure to resolve will cause CREATE UNIQUE INDEX to fail at runtime.
--
-- See also: docs/deferred-items.txt (repo root) DEFER-MFR-DUP for automation backlog.

-- Drop the unique constraint and index on code
ALTER TABLE manufacturers DROP CONSTRAINT IF EXISTS uq_manufacturers_tenant_code;
DROP INDEX IF EXISTS idx_manufacturers_code;

-- Drop columns
ALTER TABLE manufacturers DROP COLUMN IF EXISTS code;
ALTER TABLE manufacturers DROP COLUMN IF EXISTS description;
ALTER TABLE manufacturers DROP COLUMN IF EXISTS logo_url;

-- Add case-insensitive unique constraint on (tenant_id, name)
CREATE UNIQUE INDEX uq_manufacturers_tenant_name_ci ON manufacturers(tenant_id, LOWER(name));

-- Update comments
COMMENT ON TABLE manufacturers IS 'Device manufacturers for blueprint catalog (simplified: name + website only)';
