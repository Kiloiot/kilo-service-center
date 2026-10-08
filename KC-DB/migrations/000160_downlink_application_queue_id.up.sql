-- Application Center queue ids are unique per tenant, not per installation.
--
-- SCACI 3.10.1 lets the Application Center choose the queId of a dlDataQue.
-- Stored in the globally unique que_id, one tenant's id collided with another
-- tenant's row, and the resulting EEXIST disclosed that the foreign id
-- existed. que_id stays the service center's own id - drawn by the downlink
-- id allocator for every downlink and the only id a base station ever sees -
-- and ac_que_id holds the id the Application Center assigned, unique within
-- its tenant. Downlinks enqueued without an Application Center (gRPC, MQTT)
-- leave ac_que_id NULL and are reported to Application Centers by que_id.
--
-- Before this migration every row was known to Application Centers by its
-- que_id, so the backfill copies que_id into ac_que_id; each existing row
-- keeps the identity dlDataRes already reported for it. que_id was globally
-- unique, so the per-tenant index cannot conflict. The guard aborts when a
-- row carries a que_id no Application Center could have assigned.

ALTER TABLE downlink_queue
    ADD COLUMN IF NOT EXISTS ac_que_id BIGINT;

DO $$
DECLARE
    invalid bigint;
BEGIN
    SELECT count(*) INTO invalid FROM downlink_queue WHERE que_id <= 0;
    IF invalid > 0 THEN RAISE EXCEPTION 'downlink_queue.que_id is not positive on % row(s)', invalid; END IF;
END $$;

UPDATE downlink_queue SET ac_que_id = que_id WHERE ac_que_id IS NULL;

ALTER TABLE downlink_queue
    ADD CONSTRAINT downlink_queue_ac_que_id_positive CHECK (ac_que_id IS NULL OR ac_que_id > 0);

CREATE UNIQUE INDEX idx_downlink_queue_tenant_ac_que_id
    ON downlink_queue (tenant_id, ac_que_id)
    WHERE ac_que_id IS NOT NULL;
