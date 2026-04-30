-- Add connect_info column to basestation_sessions for BSSCI §5.3 compliance
-- This stores arbitrary key-value pairs from the base station's connect message

ALTER TABLE basestation_sessions
ADD COLUMN connect_info JSONB NOT NULL DEFAULT '{}'::jsonb;

COMMENT ON COLUMN basestation_sessions.connect_info IS 'BSSCI §5.3 connect arbitrary key-value pairs from base station';
