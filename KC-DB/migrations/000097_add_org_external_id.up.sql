-- Migration 000097: Add external_id column to organizations table
-- Enables OIDC external_org_claim resolution
-- The external_id maps IdP org identifiers to KiloCenter organizations

-- Add external_id column (nullable - not all orgs have external IdP mappings)
ALTER TABLE organizations
    ADD COLUMN IF NOT EXISTS external_id VARCHAR(255);

-- Create unique index for efficient lookup and enforce uniqueness
-- Only non-null values are indexed (allows multiple NULL values)
CREATE UNIQUE INDEX IF NOT EXISTS idx_organizations_external_id
    ON organizations (external_id)
    WHERE external_id IS NOT NULL;

-- Add comment for documentation
COMMENT ON COLUMN organizations.external_id IS
    'External IdP organization identifier for OIDC claim mapping';
