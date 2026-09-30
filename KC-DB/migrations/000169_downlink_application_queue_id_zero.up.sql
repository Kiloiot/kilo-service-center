-- Accept the Application Center queue id zero.
--
-- SCACI §3.10.1 (line 509) lets the Application Center assign any 64-bit
-- queId to its dlDataQue, zero included, and dlDataRes names the downlink by
-- that id (§3.12.1). The check refused zero, so such a downlink could not be
-- stored.

ALTER TABLE downlink_queue DROP CONSTRAINT downlink_queue_ac_que_id_unsigned_64;

ALTER TABLE downlink_queue
    ADD CONSTRAINT downlink_queue_ac_que_id_unsigned_64
    CHECK (ac_que_id IS NULL OR (ac_que_id >= 0 AND ac_que_id <= 18446744073709551615));
