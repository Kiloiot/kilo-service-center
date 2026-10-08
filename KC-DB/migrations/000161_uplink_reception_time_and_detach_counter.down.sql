-- Return the classifier to arrival-time windows by dropping the reception
-- time column; first_received_at still holds the arrival of every row.
-- The detach counter returns to INTEGER, wrapping values above the signed
-- range the way the 000087 down migration does.

ALTER TABLE endpoints DROP CONSTRAINT IF EXISTS chk_endpoints_last_detach_packet_cnt_unsigned;
ALTER TABLE endpoints ALTER COLUMN last_detach_packet_cnt TYPE INTEGER
  USING (CASE WHEN last_detach_packet_cnt > 2147483647 THEN last_detach_packet_cnt - 4294967296 ELSE last_detach_packet_cnt END)::INTEGER;
COMMENT ON COLUMN endpoints.last_detach_packet_cnt IS 'Endpoint packet counter from detach message';

ALTER TABLE mioty_message_deduplication DROP COLUMN IF EXISTS first_rx_time;
