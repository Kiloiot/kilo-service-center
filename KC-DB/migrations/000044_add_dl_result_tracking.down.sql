-- Remove downlink result tracking fields

-- Drop indexes
DROP INDEX IF EXISTS idx_downlink_queue_transmission_result;
DROP INDEX IF EXISTS idx_downlink_queue_pending;

-- Remove columns
ALTER TABLE downlink_queue
DROP COLUMN IF EXISTS transmission_result,
DROP COLUMN IF EXISTS transmission_time,
DROP COLUMN IF EXISTS transmission_packet_cnt;