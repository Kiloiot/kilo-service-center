-- Drop the Application Center queue id. Earlier releases report que_id to
-- Application Centers, so a downlink still in flight whose Application
-- Center id differs from its que_id would have its dlDataRes delivered under
-- an id the Application Center never assigned; the guard aborts the
-- downgrade while such a row exists. Terminal rows have reported their result
-- already and lose nothing the earlier schema could hold.

DO $$
DECLARE
    diverged bigint;
BEGIN
    SELECT count(*) INTO diverged FROM downlink_queue
    WHERE ac_que_id IS NOT NULL AND ac_que_id <> que_id
      AND status IN ('pending', 'scheduled', 'reserved', 'queued');
    IF diverged > 0 THEN RAISE EXCEPTION 'downlink_queue has % in-flight row(s) whose Application Center queue id differs from que_id', diverged; END IF;
END $$;

DROP INDEX IF EXISTS idx_downlink_queue_tenant_ac_que_id;

ALTER TABLE downlink_queue
    DROP CONSTRAINT IF EXISTS downlink_queue_ac_que_id_positive;

ALTER TABLE downlink_queue
    DROP COLUMN IF EXISTS ac_que_id;
