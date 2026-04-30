-- Base Station extensions for MIOTY protocol support
-- Based on MIOTY BS-SC Interface specification

-- Add vendor and model information
ALTER TABLE basestations 
ADD COLUMN vendor VARCHAR(100),
ADD COLUMN model VARCHAR(100),
ADD COLUMN version VARCHAR(50);

-- Add session tracking
ALTER TABLE basestations
ADD COLUMN session_uuid UUID,
ADD COLUMN session_started_at TIMESTAMP WITH TIME ZONE;

-- Add bidirectional capability
ALTER TABLE basestations
ADD COLUMN bidi BOOLEAN DEFAULT false;

-- Add operational metrics
ALTER TABLE basestations
ADD COLUMN uptime BIGINT DEFAULT 0, -- seconds since restart
ADD COLUMN ul_load REAL DEFAULT 0.0 CHECK (ul_load >= 0.0 AND ul_load <= 1.0), -- uplink channel load
ADD COLUMN dl_load REAL DEFAULT 0.0 CHECK (dl_load >= 0.0 AND dl_load <= 1.0); -- downlink channel load

-- Add status information
ALTER TABLE basestations
ADD COLUMN status_code INTEGER DEFAULT 0, -- POSIX error numbers
ADD COLUMN status_message TEXT;

-- Add index for session lookups
CREATE INDEX idx_basestations_session ON basestations(session_uuid) WHERE session_uuid IS NOT NULL;

-- Add comment documentation
COMMENT ON COLUMN basestations.vendor IS 'Base Station manufacturer name';
COMMENT ON COLUMN basestations.model IS 'Base Station hardware model';
COMMENT ON COLUMN basestations.version IS 'Base Station software version';
COMMENT ON COLUMN basestations.session_uuid IS 'Current BSSCI session identifier';
COMMENT ON COLUMN basestations.session_started_at IS 'When current session started';
COMMENT ON COLUMN basestations.bidi IS 'Bidirectional capability flag';
COMMENT ON COLUMN basestations.uptime IS 'Seconds since Base Station restart';
COMMENT ON COLUMN basestations.ul_load IS 'Uplink channel load (0.0-1.0)';
COMMENT ON COLUMN basestations.dl_load IS 'Downlink channel load (0.0-1.0)';
COMMENT ON COLUMN basestations.status_code IS 'POSIX error code for current status';
COMMENT ON COLUMN basestations.status_message IS 'Human-readable status message';
