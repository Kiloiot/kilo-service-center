-- Migration 000074 Rollback: Remove Audit Columns from DL RX Status Query Tracking

ALTER TABLE dl_rx_status_queries
    DROP COLUMN IF EXISTS bs_eui_actual,
    DROP COLUMN IF EXISTS bs_op_id_actual;
