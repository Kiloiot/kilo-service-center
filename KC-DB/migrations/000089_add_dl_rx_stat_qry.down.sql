-- Migration 000089 down: Remove dlRxStatQry field
ALTER TABLE downlink_queue DROP COLUMN IF EXISTS dl_rx_stat_qry;
