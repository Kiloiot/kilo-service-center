-- Reverse migration 039: Remove UL Data reception fields

-- Drop index
DROP INDEX IF EXISTS idx_endpoints_last_rx_time;

-- Remove columns added in migration 039
ALTER TABLE endpoints
DROP COLUMN IF EXISTS last_rx_time,
DROP COLUMN IF EXISTS last_rx_duration,
DROP COLUMN IF EXISTS packet_cnt;
