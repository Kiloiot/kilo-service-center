DROP INDEX IF EXISTS idx_basestation_sessions_active_sc_eui;

ALTER TABLE basestation_sessions DROP COLUMN IF EXISTS sc_eui;
