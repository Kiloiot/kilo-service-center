-- Migration 000142: index DL RX status query correlation and clarify spec sections.
--
-- The dlRxStat report correlation now matches tenant + endpoint + expected
-- base station + status (see MarkDLRXStatusReceived), so add a covering index
-- for that lookup. Also correct the table comments: the dlRxStatQry *query* is
-- BSSCI §5.16 while the unsolicited dlRxStat *report* is §5.15 - migration 066
-- described both as §5.15.

CREATE INDEX IF NOT EXISTS idx_dl_rx_status_queries_correlation
    ON dl_rx_status_queries (tenant_id, ep_eui, bs_eui, status, requested_at);

COMMENT ON TABLE dl_rx_status_queries IS
    'Tracks SC-initiated dlRxStatQry queries (BSSCI 5.16) awaiting the base station''s unsolicited dlRxStat report (BSSCI 5.15).';
COMMENT ON COLUMN dl_rx_status_queries.bs_eui IS
    'Expected base station EUI the query targets; a dlRxStat report is correlated to a query only when its BS EUI matches.';
