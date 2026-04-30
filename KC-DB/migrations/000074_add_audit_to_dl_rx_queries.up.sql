-- Migration 000074: Add Audit Columns to DL RX Status Query Tracking
-- Purpose: Store actual BS EUI and opId for correlation audit trail (BSSCI §5.15)
-- Rationale: BS opId lives in different namespace than SC opId - cannot correlate by opId alone
--            Correlation uses tenant_id + ep_eui (oldest pending), but we store bs_eui_actual
--            and bs_op_id_actual for audit/debugging of what the base station actually reported

-- Add audit columns to track what the base station actually sent
ALTER TABLE dl_rx_status_queries
    ADD COLUMN bs_eui_actual BYTEA CHECK (bs_eui_actual IS NULL OR length(bs_eui_actual) = 8),
    ADD COLUMN bs_op_id_actual BIGINT;

-- Documentation for audit columns
COMMENT ON COLUMN dl_rx_status_queries.bs_eui_actual IS
    'Audit field: Actual BS EUI from dlRxStat message that resolved this query. Stored for debugging but not used in correlation logic (correlation uses tenant_id + ep_eui only).';

COMMENT ON COLUMN dl_rx_status_queries.bs_op_id_actual IS
    'Audit field: Actual BS opId from dlRxStat message (positive, BS namespace per BSSCI §2.5). Stored for debugging but not used in correlation WHERE clause. BS opId ≠ SC opId - different namespaces cannot be correlated.';
