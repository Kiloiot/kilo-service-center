-- Add missing can_resume field to basestation_sessions table
-- This field is used to track whether a session can be resumed

ALTER TABLE basestation_sessions 
ADD COLUMN IF NOT EXISTS can_resume BOOLEAN DEFAULT FALSE;

-- Add comment for clarity
COMMENT ON COLUMN basestation_sessions.can_resume IS 'Whether this session can be resumed by the base station';