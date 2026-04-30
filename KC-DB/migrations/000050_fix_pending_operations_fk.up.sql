-- Migration 000050: Fix pending operations FK to match BIGINT primary key
-- The bssci_pending_operations table needs proper FK type to match basestation_sessions.id
-- basestation_sessions.id is BIGINT, not UUID

-- First drop the existing table if it exists (it's not being used due to the mismatch)
DROP TABLE IF EXISTS bssci_pending_operations;

-- Recreate with correct BIGINT foreign key
CREATE TABLE bssci_pending_operations (
    id BIGSERIAL PRIMARY KEY,
    basestation_session_id BIGINT NOT NULL REFERENCES basestation_sessions(id) ON DELETE CASCADE,
    operation_id BIGINT NOT NULL, -- MIOTY operation ID (negative for SC-initiated)
    operation_type VARCHAR(50) NOT NULL, -- attPrp, detPrp, dlDataQueue, vmActivate, etc.
    endpoint_eui BYTEA, -- For endpoint-specific operations (can be NULL for BS operations)
    operation_data JSONB NOT NULL, -- Stores the complete operation message for reissue
    metadata JSONB, -- Additional operation-specific metadata
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),

    -- Ensure unique operation IDs per session
    UNIQUE(basestation_session_id, operation_id)
);

-- Index for efficient lookups during session resume
CREATE INDEX idx_bssci_pending_operations_session ON bssci_pending_operations(basestation_session_id);
CREATE INDEX idx_bssci_pending_operations_type ON bssci_pending_operations(operation_type);
CREATE INDEX idx_bssci_pending_operations_endpoint ON bssci_pending_operations(endpoint_eui) WHERE endpoint_eui IS NOT NULL;

-- Comments for documentation
COMMENT ON TABLE bssci_pending_operations IS 'Stores incomplete BSSCI operations that must be reissued on session resume per MIOTY specification';
COMMENT ON COLUMN bssci_pending_operations.basestation_session_id IS 'BIGINT reference to basestation_sessions.id';
COMMENT ON COLUMN bssci_pending_operations.operation_id IS 'MIOTY operation ID from the original request (negative for Service Center initiated)';
COMMENT ON COLUMN bssci_pending_operations.operation_data IS 'Complete MessagePack operation message that can be directly reissued';
COMMENT ON COLUMN bssci_pending_operations.endpoint_eui IS 'Target endpoint EUI for endpoint-specific operations like attach/detach propagate';