-- Migration 000089: Add dlRxStatQry field per SCACI §3.10.1
-- This optional field allows Application Centers to request DL RX status info from endpoints

ALTER TABLE downlink_queue ADD COLUMN dl_rx_stat_qry BOOLEAN NOT NULL DEFAULT false;

COMMENT ON COLUMN downlink_queue.dl_rx_stat_qry IS 'SCACI §3.10.1: True to query DL RX status from endpoint';
