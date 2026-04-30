ALTER TABLE organizations
  DROP COLUMN IF EXISTS description,
  DROP COLUMN IF EXISTS can_have_base_stations,
  DROP COLUMN IF EXISTS max_base_station_count,
  DROP COLUMN IF EXISTS max_endpoint_count,
  DROP COLUMN IF EXISTS tags;
DROP INDEX IF EXISTS idx_organizations_tags;
