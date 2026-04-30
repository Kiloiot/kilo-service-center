-- Rollback MIOTY Base Station Status Response Fields migration
-- This removes the statusRsp fields added in the up migration

-- Drop indexes first
DROP INDEX IF EXISTS idx_basestations_system_health;
DROP INDEX IF EXISTS idx_basestations_last_status;

-- Remove status response fields from basestations table
ALTER TABLE basestations
DROP COLUMN IF EXISTS last_status_at,
DROP COLUMN IF EXISTS bs_config,
DROP COLUMN IF EXISTS memory_load,
DROP COLUMN IF EXISTS cpu_load,
DROP COLUMN IF EXISTS temperature_celsius,
DROP COLUMN IF EXISTS uptime_seconds,
DROP COLUMN IF EXISTS duty_cycle,
DROP COLUMN IF EXISTS system_time;