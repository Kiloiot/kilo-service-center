-- Reverse Base Station MIOTY compliance fixes

-- Drop added fields
ALTER TABLE basestations 
DROP COLUMN IF EXISTS info,
DROP COLUMN IF EXISTS hardware_name;

-- Rename sw_version back to version
ALTER TABLE basestations RENAME COLUMN sw_version TO version;

-- Drop indexes
DROP INDEX IF EXISTS idx_basestations_hardware_name;
DROP INDEX IF EXISTS idx_basestations_info;