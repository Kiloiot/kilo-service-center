-- Add MIOTY-specific fields to endpoints table
-- Based on MIOTY EP (End Point) specifications

-- Add bidirectional capability
ALTER TABLE endpoints 
ADD COLUMN bidi BOOLEAN DEFAULT false;

-- Add pre-attachment mode
ALTER TABLE endpoints
ADD COLUMN pre_attach BOOLEAN DEFAULT false;

-- Add short address assigned by Base Station
ALTER TABLE endpoints
ADD COLUMN sh_addr INTEGER;

-- Add attachment counter
ALTER TABLE endpoints
ADD COLUMN attach_cnt INTEGER DEFAULT 0;

-- Add current packet counter
ALTER TABLE endpoints
ADD COLUMN packet_cnt INTEGER DEFAULT 0;

-- Add channel configuration
ALTER TABLE endpoints
ADD COLUMN dual_chan BOOLEAN DEFAULT false;

-- Add downlink repetition count
ALTER TABLE endpoints
ADD COLUMN repetition SMALLINT DEFAULT 1 CHECK (repetition >= 1 AND repetition <= 15);

-- Add wide carrier offset flag
ALTER TABLE endpoints
ADD COLUMN wide_carr_off BOOLEAN DEFAULT false;

-- Add long downlink interblock distance
ALTER TABLE endpoints
ADD COLUMN long_blk_dist BOOLEAN DEFAULT false;

-- Add endpoint status
ALTER TABLE endpoints
ADD COLUMN ep_status VARCHAR(20) DEFAULT 'detached' CHECK (ep_status IN ('attached', 'detached', 'attaching'));

-- Add index for status queries
CREATE INDEX idx_endpoints_status ON endpoints(tenant_id, ep_status) WHERE ep_status = 'attached';

-- Add comment documentation
COMMENT ON COLUMN endpoints.bidi IS 'Bidirectional capability flag';
COMMENT ON COLUMN endpoints.pre_attach IS 'Pre-attachment mode enabled';
COMMENT ON COLUMN endpoints.sh_addr IS 'Short address assigned by Base Station';
COMMENT ON COLUMN endpoints.attach_cnt IS 'Attachment counter';
COMMENT ON COLUMN endpoints.packet_cnt IS 'Current packet counter';
COMMENT ON COLUMN endpoints.dual_chan IS 'Dual channel configuration';
COMMENT ON COLUMN endpoints.repetition IS 'Downlink repetition count (1-15)';
COMMENT ON COLUMN endpoints.wide_carr_off IS 'Wide carrier offset flag';
COMMENT ON COLUMN endpoints.long_blk_dist IS 'Long downlink interblock distance';
COMMENT ON COLUMN endpoints.ep_status IS 'End Point attachment status';
