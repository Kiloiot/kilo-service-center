-- Add missing MIOTY attach propagate fields per BSSCI v1.0.0 specification
ALTER TABLE endpoints 
ADD COLUMN IF NOT EXISTS dual_chan BOOLEAN DEFAULT false,
ADD COLUMN IF NOT EXISTS repetition BOOLEAN DEFAULT true,
ADD COLUMN IF NOT EXISTS wide_carr_off BOOLEAN DEFAULT false,
ADD COLUMN IF NOT EXISTS long_blk_dist BOOLEAN DEFAULT false,
ADD COLUMN IF NOT EXISTS last_packet_cnt INTEGER DEFAULT 0;

-- Add comments for documentation
COMMENT ON COLUMN endpoints.dual_chan IS 'True if End Point uses dual channel mode';
COMMENT ON COLUMN endpoints.repetition IS 'True if End Point uses DL repetition';
COMMENT ON COLUMN endpoints.wide_carr_off IS 'True if End Point uses wide carrier offset';
COMMENT ON COLUMN endpoints.long_blk_dist IS 'True if End Point uses long DL interblock distance';
COMMENT ON COLUMN endpoints.last_packet_cnt IS 'Last known End Point packet counter';