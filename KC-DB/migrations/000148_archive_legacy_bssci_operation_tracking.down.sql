-- Recreate the legacy bssci_operation_tracking table as migration 028 defined
-- it. The archived system_events rows stay in place and the table comes back
-- empty: the legacy rows never carried the operation payload, so nothing is
-- reconstructed from the archive.

CREATE TABLE IF NOT EXISTS bssci_operation_tracking (
    id BIGSERIAL PRIMARY KEY,
    session_id BIGINT NOT NULL REFERENCES basestation_sessions(id) ON DELETE CASCADE,
    op_id BIGINT NOT NULL,
    command VARCHAR(50) NOT NULL,
    direction VARCHAR(10) NOT NULL CHECK (direction IN ('inbound', 'outbound')),
    state VARCHAR(20) NOT NULL DEFAULT 'pending' CHECK (state IN ('pending', 'acknowledged', 'completed', 'failed')),
    initiated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    acknowledged_at TIMESTAMPTZ,
    completed_at TIMESTAMPTZ,
    response_data JSONB,
    error_message TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_bssci_operation_session ON bssci_operation_tracking(session_id, op_id);
CREATE INDEX IF NOT EXISTS idx_bssci_operation_pending ON bssci_operation_tracking(session_id, state) WHERE state = 'pending';

COMMENT ON TABLE bssci_operation_tracking IS 'Tracks BSSCI operations for session resumption per MIOTY spec';
