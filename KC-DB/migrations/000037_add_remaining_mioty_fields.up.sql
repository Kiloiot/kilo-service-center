-- Add remaining MIOTY endpoint fields per BSSCI v1.0.0 specification
-- These fields are used in Attach and Attach Propagate operations

-- Add missing MIOTY configuration fields per BSSCI v1.0.0 Section 3.8.1 (Attach Propagate)
ALTER TABLE endpoints
ADD COLUMN IF NOT EXISTS dual_chan BOOLEAN DEFAULT false,     -- True if End Point uses dual channel mode
ADD COLUMN IF NOT EXISTS wide_carr_off BOOLEAN DEFAULT false, -- True if End Point uses wide carrier offset
ADD COLUMN IF NOT EXISTS long_blk_dist BOOLEAN DEFAULT false, -- True if End Point uses long block distance
ADD COLUMN IF NOT EXISTS last_packet_cnt INTEGER DEFAULT 0;   -- Last known packet counter for deduplication

-- Add comments per MIOTY specification
COMMENT ON COLUMN endpoints.dual_chan IS 'True if End Point uses dual channel mode per MIOTY BSSCI v1.0.0 Section 3.8.1';
COMMENT ON COLUMN endpoints.wide_carr_off IS 'True if End Point uses wide carrier offset per MIOTY BSSCI v1.0.0 Section 3.8.1';
COMMENT ON COLUMN endpoints.long_blk_dist IS 'True if End Point uses long block distance per MIOTY BSSCI v1.0.0 Section 3.8.1';
COMMENT ON COLUMN endpoints.last_packet_cnt IS 'Last known packet counter from End Point for deduplication per MIOTY BSSCI v1.0.0 Section 3.8.1';

-- Add index for packet counter deduplication queries
CREATE INDEX IF NOT EXISTS idx_endpoints_packet_cnt
ON endpoints(ep_eui, last_packet_cnt)
WHERE last_packet_cnt > 0;

-- Update existing endpoints with default values
UPDATE endpoints 
SET dual_chan = false, 
    wide_carr_off = false, 
    long_blk_dist = false, 
    last_packet_cnt = 0
WHERE dual_chan IS NULL 
   OR wide_carr_off IS NULL 
   OR long_blk_dist IS NULL 
   OR last_packet_cnt IS NULL;