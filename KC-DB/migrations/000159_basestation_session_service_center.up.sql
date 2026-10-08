-- The service center (its EUI) that owns a BSSCI session row. A restarting
-- service center hands the sessions a previous process of its own left
-- active back resumable; rows of other service centers are not touched.
ALTER TABLE basestation_sessions ADD COLUMN IF NOT EXISTS sc_eui BYTEA;

CREATE INDEX IF NOT EXISTS idx_basestation_sessions_active_sc_eui
    ON basestation_sessions (sc_eui)
    WHERE status = 'active';
