DROP INDEX IF EXISTS idx_org_members_permissions;
ALTER TABLE organization_members
  DROP COLUMN IF EXISTS is_org_admin,
  DROP COLUMN IF EXISTS is_base_station_admin,
  DROP COLUMN IF EXISTS is_endpoint_admin;
