-- Migration 000142 down: drop the correlation index and restore the prior
-- (less precise) table comment. The bs_eui column comment is left in place; it
-- documents an existing column accurately.
DROP INDEX IF EXISTS idx_dl_rx_status_queries_correlation;

COMMENT ON TABLE dl_rx_status_queries IS
    'Tracks DL RX status queries (BSSCI 5.15) and their responses.';
