-- Remove Service Center session UUID field from basestation_sessions table

ALTER TABLE basestation_sessions 
DROP COLUMN IF EXISTS sn_sc_uuid;