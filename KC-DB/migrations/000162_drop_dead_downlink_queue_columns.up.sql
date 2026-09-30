-- Drop the downlink_queue columns no repository reads or writes, with the
-- index that led on one of them.
--
-- prio (000032): the dispatcher and every listing order by priority; prio was
--   never written after the column was added, so it only ever held 0.0.
-- valid_until (001): downlink expiry is reported by the base station in
--   dlDataRes (BSSCI 3.14); the column was never assigned.
-- packet_cnt_array (028): counter dependent downlinks store their packet
--   counters in packet_cnt; the array column was never assigned.
--
-- idx_downlink_queue_mioty_prio (000151) indexed (status, prio DESC,
-- created_at); with prio constant it served no query.
--
-- Each column is guarded by its own assertion for non-default data; any
-- populated value aborts the migration for operator review.

DO $$
DECLARE
    populated bigint;
BEGIN
    SELECT count(*) INTO populated FROM downlink_queue WHERE prio IS NOT NULL AND prio <> 0.0;
    IF populated > 0 THEN RAISE EXCEPTION 'downlink_queue.prio carries a non-default value on % row(s)', populated; END IF;

    SELECT count(*) INTO populated FROM downlink_queue WHERE valid_until IS NOT NULL;
    IF populated > 0 THEN RAISE EXCEPTION 'downlink_queue.valid_until is populated on % row(s)', populated; END IF;

    SELECT count(*) INTO populated FROM downlink_queue WHERE packet_cnt_array IS NOT NULL;
    IF populated > 0 THEN RAISE EXCEPTION 'downlink_queue.packet_cnt_array is populated on % row(s)', populated; END IF;
END $$;

DROP INDEX IF EXISTS idx_downlink_queue_mioty_prio;

ALTER TABLE downlink_queue
    DROP COLUMN IF EXISTS prio,
    DROP COLUMN IF EXISTS valid_until,
    DROP COLUMN IF EXISTS packet_cnt_array;
