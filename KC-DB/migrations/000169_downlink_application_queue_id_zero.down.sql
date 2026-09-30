-- Refuse the Application Center queue id zero again. The guard aborts while a
-- row holds it, which the restored check would reject.

DO $$
DECLARE
    zero_ids bigint;
BEGIN
    SELECT count(*) INTO zero_ids FROM downlink_queue WHERE ac_que_id = 0;
    IF zero_ids > 0 THEN RAISE EXCEPTION 'downlink_queue.ac_que_id is zero on % row(s)', zero_ids; END IF;
END $$;

ALTER TABLE downlink_queue DROP CONSTRAINT downlink_queue_ac_que_id_unsigned_64;

ALTER TABLE downlink_queue
    ADD CONSTRAINT downlink_queue_ac_que_id_unsigned_64
    CHECK (ac_que_id IS NULL OR (ac_que_id > 0 AND ac_que_id <= 18446744073709551615));
