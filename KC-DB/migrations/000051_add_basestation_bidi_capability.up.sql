-- Migration 000051: Add bidirectional capability to basestations
-- Store the base station's hardware capability to transmit downlink messages
-- This is reported during BSSCI Connect and must persist across sessions

-- Add bidi column to basestations table
ALTER TABLE basestations
ADD COLUMN IF NOT EXISTS bidi BOOLEAN DEFAULT true;

COMMENT ON COLUMN basestations.bidi IS 'Hardware capability to transmit downlink (from BSSCI Connect message)';

-- Also add to basestation_sessions for session-specific tracking
ALTER TABLE basestation_sessions
ADD COLUMN IF NOT EXISTS bidi BOOLEAN DEFAULT true;

COMMENT ON COLUMN basestation_sessions.bidi IS 'Base station bidirectional capability for this session';

-- Index for filtering bidirectional-capable base stations
CREATE INDEX IF NOT EXISTS idx_basestations_bidi ON basestations(bidi) WHERE bidi = true;