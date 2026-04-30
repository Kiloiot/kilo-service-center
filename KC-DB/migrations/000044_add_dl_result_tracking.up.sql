-- Add downlink result tracking fields per MIOTY BSSCI v1.0.0 Section 5.14
-- This migration adds fields to track the transmission result of queued downlink messages

-- Add transmission result tracking fields to downlink_queue table
ALTER TABLE downlink_queue
ADD COLUMN IF NOT EXISTS transmission_result VARCHAR(20),
ADD COLUMN IF NOT EXISTS transmission_time BIGINT,
ADD COLUMN IF NOT EXISTS transmission_packet_cnt BIGINT;

-- Add comments for MIOTY compliance documentation
COMMENT ON COLUMN downlink_queue.transmission_result IS 'Result of downlink transmission: sent, expired, invalid, or revoked per MIOTY BSSCI 5.14.1';
COMMENT ON COLUMN downlink_queue.transmission_time IS 'Unix UTC time of transmission, center of first subpacket, 64 bit, ns resolution per MIOTY BSSCI 5.14.1';
COMMENT ON COLUMN downlink_queue.transmission_packet_cnt IS 'End Point packet counter when transmitted per MIOTY BSSCI 5.14.1';

-- Add index for querying transmission results
CREATE INDEX IF NOT EXISTS idx_downlink_queue_transmission_result 
ON downlink_queue(tenant_id, transmission_result, transmission_time DESC) 
WHERE transmission_result IS NOT NULL;

-- Add index for finding pending transmissions
CREATE INDEX IF NOT EXISTS idx_downlink_queue_pending 
ON downlink_queue(tenant_id, que_id) 
WHERE transmission_result IS NULL;