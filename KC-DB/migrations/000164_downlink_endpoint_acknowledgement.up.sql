-- Record when an endpoint acknowledged a downlink.
--
-- An uplink carrying dlAck acknowledges the downlink transmitted in the
-- window of the previous packet counter (BSSCI §3.10.1, radio §3.6.5.1 ACK
-- flag). endpoint_acked_at stores the time that acknowledgement arrived on the
-- transmitted row whose transmission_packet_cnt is that counter; it stays NULL
-- for a downlink the endpoint has not acknowledged. Existing rows carry no
-- acknowledgement, so nothing is backfilled.

ALTER TABLE downlink_queue ADD COLUMN IF NOT EXISTS endpoint_acked_at TIMESTAMPTZ;

COMMENT ON COLUMN downlink_queue.endpoint_acked_at IS
    'Time an uplink with dlAck acknowledged this transmitted downlink (BSSCI §3.10.1 dlAck, window of packetCnt - 1); NULL until acknowledged';

CREATE INDEX IF NOT EXISTS idx_downlink_queue_ack_window
    ON downlink_queue (tenant_id, ep_eui, transmission_packet_cnt)
    WHERE status = 'transmitted' AND endpoint_acked_at IS NULL;
