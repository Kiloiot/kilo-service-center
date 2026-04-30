ALTER TABLE basestations
  DROP COLUMN IF EXISTS location_source,
  DROP COLUMN IF EXISTS location_updated_at;
