-- Fix Base Station MIOTY compliance per BSSCI v1.0.0 specification
-- This addresses field naming and missing fields issues

-- Rename 'version' to 'sw_version' to match MIOTY spec field name
ALTER TABLE basestations RENAME COLUMN version TO sw_version;

-- Add missing 'info' field for additional base station information object
-- This allows arbitrary key-value pairs as per MIOTY spec
ALTER TABLE basestations 
ADD COLUMN IF NOT EXISTS info JSONB DEFAULT '{}';

-- Add hardware_name field to distinguish from user-provided name
-- MIOTY spec 'name' field is hardware-provided name from base station
ALTER TABLE basestations
ADD COLUMN IF NOT EXISTS hardware_name VARCHAR(255);

-- Update comments for MIOTY compliance
COMMENT ON COLUMN basestations.sw_version IS 'MIOTY swVersion field - Software version from base station';
COMMENT ON COLUMN basestations.info IS 'MIOTY info field - Additional base station information object with arbitrary key-value pairs';
COMMENT ON COLUMN basestations.hardware_name IS 'MIOTY name field - Hardware-provided name from base station (distinct from user name)';
COMMENT ON COLUMN basestations.name IS 'User-provided display name for UI (not part of MIOTY spec)';

-- Add index for hardware name lookups
CREATE INDEX IF NOT EXISTS idx_basestations_hardware_name ON basestations(hardware_name) 
WHERE hardware_name IS NOT NULL;

-- Add index for info field queries
CREATE INDEX IF NOT EXISTS idx_basestations_info ON basestations USING gin(info)
WHERE info != '{}';