-- Return ac_que_id to BIGINT. The guard aborts while a row holds an
-- Application Center queue id at or above 2^63, which BIGINT cannot store.

DO $$
DECLARE
    beyond bigint;
BEGIN
    SELECT count(*) INTO beyond FROM downlink_queue WHERE ac_que_id > 9223372036854775807;
    IF beyond > 0 THEN RAISE EXCEPTION 'downlink_queue.ac_que_id exceeds BIGINT on % row(s)', beyond; END IF;
END $$;

ALTER TABLE downlink_queue DROP CONSTRAINT downlink_queue_ac_que_id_unsigned_64;

ALTER TABLE downlink_queue ALTER COLUMN ac_que_id TYPE BIGINT;

ALTER TABLE downlink_queue
    ADD CONSTRAINT downlink_queue_ac_que_id_positive CHECK (ac_que_id IS NULL OR ac_que_id > 0);
