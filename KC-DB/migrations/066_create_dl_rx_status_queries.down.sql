-- Migration 066 Rollback: Remove DL RX Status Query Tracking Table
-- Reverses: 066_create_dl_rx_status_queries.up.sql

DROP INDEX IF EXISTS idx_dl_rx_status_queries_status_requested;
DROP INDEX IF EXISTS idx_dl_rx_status_queries_tenant_ep_op;
DROP TABLE IF EXISTS dl_rx_status_queries;
