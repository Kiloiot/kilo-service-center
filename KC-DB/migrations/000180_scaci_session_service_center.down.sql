DROP INDEX IF EXISTS idx_scaci_sessions_live_sc_eui;

ALTER TABLE scaci_sessions DROP COLUMN IF EXISTS sc_eui;
