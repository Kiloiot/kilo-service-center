-- Migration 066: Create DL RX Status Query Tracking Table
-- Purpose: Track dlRxStatQry → dlRxStat correlation for BSSCI §5.15 audit trail
-- Related: DEFER-008 (background cleanup job with 15-minute timeout)

CREATE TABLE IF NOT EXISTS dl_rx_status_queries (
    id BIGSERIAL PRIMARY KEY,
    tenant_id BIGINT NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    ep_eui BYTEA NOT NULL CHECK (length(ep_eui) = 8),
    bs_eui BYTEA NOT NULL CHECK (length(bs_eui) = 8),
    op_id BIGINT NOT NULL, -- Negative SC opId per BSSCI §2.5
    status TEXT NOT NULL CHECK (status IN ('pending', 'received', 'timeout')),
    requested_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    received_at TIMESTAMPTZ
);

-- Ensure unique tracking per tenant/endpoint/operation
CREATE UNIQUE INDEX idx_dl_rx_status_queries_tenant_ep_op
    ON dl_rx_status_queries(tenant_id, ep_eui, op_id);

-- Support efficient cleanup queries for DEFER-008 background job
CREATE INDEX idx_dl_rx_status_queries_status_requested
    ON dl_rx_status_queries(status, requested_at);

-- Documentation
COMMENT ON TABLE dl_rx_status_queries IS
    'Tracks dlRxStatQry requests to correlate with dlRxStat responses per BSSCI §5.15. Queries remain pending until dlRxStat arrives or DEFER-008 cleanup job marks timeout (15 minutes).';

COMMENT ON COLUMN dl_rx_status_queries.op_id IS
    'Service Center operation ID (negative per BSSCI §2.5) from dlRxStatQry request.';

COMMENT ON COLUMN dl_rx_status_queries.status IS
    'Query lifecycle: pending (awaiting dlRxStat), received (dlRxStat arrived), timeout (DEFER-008 cleanup marked expired after 15 minutes).';

COMMENT ON COLUMN dl_rx_status_queries.requested_at IS
    'Timestamp when dlRxStatQry was sent to base station. Used by DEFER-008 cleanup job to detect queries older than 15 minutes.';

COMMENT ON COLUMN dl_rx_status_queries.received_at IS
    'Timestamp when corresponding dlRxStat was received, or when dlRxStatQryCmp completed. NULL for pending/timeout queries.';
