-- Remove can_resume field from basestation_sessions table

ALTER TABLE basestation_sessions 
DROP COLUMN IF EXISTS can_resume;