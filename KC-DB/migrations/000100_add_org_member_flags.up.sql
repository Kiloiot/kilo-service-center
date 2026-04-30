-- Add boolean permission flags to organization_members
ALTER TABLE organization_members
  ADD COLUMN IF NOT EXISTS is_org_admin BOOLEAN NOT NULL DEFAULT false,
  ADD COLUMN IF NOT EXISTS is_base_station_admin BOOLEAN NOT NULL DEFAULT false,
  ADD COLUMN IF NOT EXISTS is_endpoint_admin BOOLEAN NOT NULL DEFAULT false;

-- Backfill: existing owner/admin roles get is_org_admin=true
-- NOTE: SQL migrations cannot use Go constants. These inline literals ('owner', 'admin', 'active')
-- are an ALLOWED EXCEPTION documented in:
--   - docs/constants-inventory.md (repo root, SQL Migration Exemptions section)
--   - compliance/constants/KC-DB-constants.md (repo root, Migration 000100 exemption)
-- Values MUST match organization_constants.go: OrganizationRoleOwner, OrganizationRoleAdmin, OrganizationMemberStatusActive
UPDATE organization_members
SET is_org_admin = true
WHERE role IN ('owner', 'admin') AND status = 'active';

CREATE INDEX IF NOT EXISTS idx_org_members_permissions
  ON organization_members (org_id, user_id, is_org_admin, is_base_station_admin, is_endpoint_admin)
  WHERE status = 'active';

COMMENT ON COLUMN organization_members.is_org_admin IS 'Can manage org settings and users';
COMMENT ON COLUMN organization_members.is_base_station_admin IS 'Can manage base stations for this org';
COMMENT ON COLUMN organization_members.is_endpoint_admin IS 'Can manage endpoints for this org';
