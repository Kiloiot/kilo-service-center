-- Add encoding column to basestation_sessions table
-- This supports BSSCI Section 1 requirement for JSON/MessagePack dual encoding
-- Each session negotiates its encoding on first message and persists it for resume scenarios

-- Add encoding column with default='msgpack' to maintain backwards compatibility
ALTER TABLE basestation_sessions
ADD COLUMN IF NOT EXISTS encoding VARCHAR(10) NOT NULL DEFAULT 'msgpack';

COMMENT ON COLUMN basestation_sessions.encoding IS 'Message encoding for this session (json or msgpack) per BSSCI Section 1 - negotiated on first message';

-- Add index for encoding-based queries (useful for debugging/monitoring)
CREATE INDEX IF NOT EXISTS idx_basestation_sessions_encoding
ON basestation_sessions(encoding);

-- Update existing sessions to have explicit encoding value
-- (They were using msgpack implicitly before this migration)
UPDATE basestation_sessions
SET encoding = 'msgpack'
WHERE encoding IS NULL OR encoding = '';
