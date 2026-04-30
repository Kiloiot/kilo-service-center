-- Add description, quota fields, and tags to organizations
-- NOTE: TEXT columns default to NULL, no explicit DEFAULT NULL needed
ALTER TABLE organizations
  ADD COLUMN IF NOT EXISTS description TEXT,
  ADD COLUMN IF NOT EXISTS can_have_base_stations BOOLEAN NOT NULL DEFAULT true,
  ADD COLUMN IF NOT EXISTS max_base_station_count INTEGER,
  ADD COLUMN IF NOT EXISTS max_endpoint_count INTEGER,
  ADD COLUMN IF NOT EXISTS tags HSTORE DEFAULT ''::hstore;

CREATE INDEX IF NOT EXISTS idx_organizations_tags ON organizations USING gin(tags);

COMMENT ON COLUMN organizations.description IS 'Organization description (nullable)';
COMMENT ON COLUMN organizations.can_have_base_stations IS 'Whether org can own base stations';
COMMENT ON COLUMN organizations.max_base_station_count IS 'NULL = unlimited';
COMMENT ON COLUMN organizations.max_endpoint_count IS 'NULL = unlimited';
COMMENT ON COLUMN organizations.tags IS 'Arbitrary key-value metadata (HSTORE)';
