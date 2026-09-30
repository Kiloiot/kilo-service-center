-- Drop the endpoint acknowledgement time. Earlier releases never read it, so
-- only the record of which transmitted downlinks an endpoint acknowledged is
-- lost; the downlinks and their transmission results stay.

DROP INDEX IF EXISTS idx_downlink_queue_ack_window;

ALTER TABLE downlink_queue DROP COLUMN IF EXISTS endpoint_acked_at;
