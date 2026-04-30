-- Migration 000049: Add pending operations table for MIOTY BSSCI v1.0.0 session resume compliance
-- This table stores incomplete operations that must be reissued when sessions are resumed per MIOTY spec Section 1

CREATE TABLE IF NOT EXISTS bssci_pending_operations (
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
COMMENT ON COLUMN bssci_pending_operations.operation_id IS 'MIOTY operation ID from the original request (negative for Service Center initiated)';
COMMENT ON COLUMN bssci_pending_operations.operation_data IS 'Complete MessagePack operation message that can be directly reissued';
COMMENT ON COLUMN bssci_pending_operations.endpoint_eui IS 'Target endpoint EUI for endpoint-specific operations like attach/detach propagate';