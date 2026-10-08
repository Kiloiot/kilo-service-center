-- Classify uplinks by the radio reception time instead of service center
-- arrival.
--
-- Every base station stamps the telegram's own radio time (BSSCI §3.10.1
-- rxTime), so a ulData reissued after a session resume or delivered late
-- by a station belongs to the same telegram however long after the first
-- reception it reaches the service center. first_rx_time records the radio
-- time of the reception that created the classifier row; first_received_at
-- and last_received_at keep the arrival times for operators.
--
-- Existing rows take the reception time of the message they point to. The
-- foreign key on first_message_id guarantees that message exists; the
-- arrival time in nanoseconds is the fallback that keeps NOT NULL safe.

ALTER TABLE mioty_message_deduplication ADD COLUMN first_rx_time BIGINT;

UPDATE mioty_message_deduplication d
SET first_rx_time = COALESCE(
    (SELECT m.rx_time FROM messages m WHERE m.id = d.first_message_id),
    (EXTRACT(EPOCH FROM d.first_received_at) * 1000000000)::BIGINT
);

ALTER TABLE mioty_message_deduplication ALTER COLUMN first_rx_time SET NOT NULL;

COMMENT ON COLUMN mioty_message_deduplication.first_rx_time IS
    'Radio reception time (Unix UTC ns, BSSCI §3.10.1 rxTime) of the reception that created the row; the duplicate window is measured from it';

-- endpoints.last_detach_packet_cnt stores the 32-bit packet counter of a
-- detach (BSSCI §3.7.1); 000087 widened the other endpoint counters to BIGINT
-- and left this one INTEGER, so a counter above 2147483647 could not be
-- recorded. The endpoints table is small, so the rewrite is cheap.
ALTER TABLE endpoints ALTER COLUMN last_detach_packet_cnt TYPE BIGINT USING last_detach_packet_cnt::BIGINT;
UPDATE endpoints SET last_detach_packet_cnt = last_detach_packet_cnt + 4294967296 WHERE last_detach_packet_cnt < 0;
ALTER TABLE endpoints ADD CONSTRAINT chk_endpoints_last_detach_packet_cnt_unsigned
    CHECK (last_detach_packet_cnt IS NULL OR (last_detach_packet_cnt >= 0 AND last_detach_packet_cnt <= 4294967295));
COMMENT ON COLUMN endpoints.last_detach_packet_cnt IS 'Packet counter of the last detach (uint32, 0-4294967295) per BSSCI §3.7.1';
