-- Add missing UL Data reception fields per MIOTY BSSCI v1.0.0 Section 5.10.1
-- These fields are updated for every UL Data message received from endpoints

-- Add UL Data reception tracking fields to endpoints table
ALTER TABLE endpoints
ADD COLUMN IF NOT EXISTS last_rx_time BIGINT,          -- rxTime: Unix UTC time of last UL Data reception (ns)
ADD COLUMN IF NOT EXISTS last_rx_duration BIGINT,      -- rxDuration: Duration of last UL Data reception (ns)
ADD COLUMN IF NOT EXISTS packet_cnt INTEGER DEFAULT 0; -- packetCnt: Current End Point packet counter

-- Add comments per MIOTY specification
COMMENT ON COLUMN endpoints.last_rx_time IS 'Unix UTC time of last UL Data reception, center of last subpacket, 64 bit, ns resolution per MIOTY BSSCI v1.0.0 Section 5.10.1';
COMMENT ON COLUMN endpoints.last_rx_duration IS 'Duration of last UL Data reception, center of first to last subpacket in ns per MIOTY BSSCI v1.0.0 Section 5.10.1';
COMMENT ON COLUMN endpoints.packet_cnt IS 'Current End Point packet counter from last UL Data per MIOTY BSSCI v1.0.0 Section 5.10.1';

-- Add index for reception time queries
CREATE INDEX IF NOT EXISTS idx_endpoints_last_rx_time 
ON endpoints(tenant_id, last_rx_time DESC) 
WHERE last_rx_time IS NOT NULL;

-- Update existing endpoints with default values
UPDATE endpoints 
SET packet_cnt = 0
WHERE packet_cnt IS NULL;