ALTER TABLE basestations
  ADD COLUMN IF NOT EXISTS location_source VARCHAR(10) DEFAULT NULL CHECK (location_source IN ('gps', 'manual')),
  ADD COLUMN IF NOT EXISTS location_updated_at TIMESTAMPTZ DEFAULT NULL;
