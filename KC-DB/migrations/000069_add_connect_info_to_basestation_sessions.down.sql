-- Remove connect_info column from basestation_sessions

ALTER TABLE basestation_sessions
DROP COLUMN connect_info;
