-- Rollback: restore the session_id column and index
-- Note: This column was not used in production code, so restoration is for schema compatibility only.

ALTER TABLE basestation_sessions
ADD COLUMN IF NOT EXISTS session_id UUID DEFAULT gen_random_uuid();

-- Recreate the index
CREATE INDEX IF NOT EXISTS idx_basestation_sessions_session ON basestation_sessions(session_id);
