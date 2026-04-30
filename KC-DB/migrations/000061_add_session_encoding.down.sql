-- Rollback: Remove encoding column from basestation_sessions table

-- Drop the index first
DROP INDEX IF EXISTS idx_basestation_sessions_encoding;

-- Remove the encoding column
ALTER TABLE basestation_sessions
DROP COLUMN IF EXISTS encoding;
