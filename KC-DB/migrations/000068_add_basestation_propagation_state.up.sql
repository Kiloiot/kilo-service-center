-- Migration 068: Add basestation propagation state table for automatic endpoint propagation reconciliation
-- BSSCI §5.8-5.8.3: Automatic attach/detach propagation across base stations
-- This table tracks per-base-station progress for automatic propagation when base stations connect
--
-- Related application-layer changes (no schema impact):
--   - Network session keys now encrypted before storage (BSSCI §5.6.2)
--   - Encryption implemented in KC-Core/pkg/bssci/server.go:4230-4242
--   - SendAttachPropagate calls keyEncryptor.EncryptKeyRaw(nwkSnKey)

CREATE TABLE IF NOT EXISTS basestation_propagation_state (
    base_station_id BIGINT PRIMARY KEY REFERENCES basestations(id) ON DELETE CASCADE,
    last_full_sync_at TIMESTAMPTZ,
    last_endpoint_cursor BIGINT DEFAULT 0,  -- Last processed endpoint.id (global across tenants)
    status TEXT NOT NULL DEFAULT 'idle',    -- idle, running, completed, error
    retry_count INT DEFAULT 0,
    next_retry_at TIMESTAMPTZ,              -- For exponential backoff
    last_error TEXT,
    tenant_progress JSONB DEFAULT '{}'::JSONB,  -- Per-tenant observability metrics
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX idx_propagation_state_status ON basestation_propagation_state(status);
CREATE INDEX idx_propagation_state_retry ON basestation_propagation_state(next_retry_at) WHERE status = 'error';

COMMENT ON TABLE basestation_propagation_state IS 'Tracks automatic propagation progress per base station for cross-tenant roaming';
COMMENT ON COLUMN basestation_propagation_state.base_station_id IS 'Base station ID - single row per BS, tracks global progress across all tenants';
COMMENT ON COLUMN basestation_propagation_state.last_endpoint_cursor IS 'Last processed endpoint.id for resume after interruption';
COMMENT ON COLUMN basestation_propagation_state.tenant_progress IS 'JSONB map: {"tenant_<id>": {"endpoints_processed": N, "last_updated": "timestamp", "errors": []}}';
COMMENT ON COLUMN basestation_propagation_state.next_retry_at IS 'Scheduled retry time for exponential backoff after errors';
