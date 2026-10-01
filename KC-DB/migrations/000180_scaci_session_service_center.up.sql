-- The service center (its EUI) that owns a SCACI session row. A restarting
-- service center hands the sessions a previous process of its own left live
-- back to the resumable disconnected state; rows of other service centers are
-- not touched.
ALTER TABLE scaci_sessions ADD COLUMN IF NOT EXISTS sc_eui BYTEA;

CREATE INDEX IF NOT EXISTS idx_scaci_sessions_live_sc_eui
    ON scaci_sessions (sc_eui)
    WHERE status IN ('active', 'resumed');
