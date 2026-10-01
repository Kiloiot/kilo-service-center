-- Hold the Application Center's queue id over its full unsigned 64-bit range.
--
-- SCACI §3.10.1 defines the queId of a dlDataQue as a 64-bit numeric that the
-- Application Center chooses, and dlDataRes must name the downlink by that id
-- (§3.12.1). The BIGINT column refused every id at or above 2^63. NUMERIC(20,0)
-- stores the id itself, so the value an operator reads is the value the
-- Application Center sent; the per-tenant unique index is rebuilt with it.

ALTER TABLE downlink_queue DROP CONSTRAINT downlink_queue_ac_que_id_positive;

ALTER TABLE downlink_queue ALTER COLUMN ac_que_id TYPE NUMERIC(20, 0);

ALTER TABLE downlink_queue
    ADD CONSTRAINT downlink_queue_ac_que_id_unsigned_64
    CHECK (ac_que_id IS NULL OR (ac_que_id > 0 AND ac_que_id <= 18446744073709551615));
