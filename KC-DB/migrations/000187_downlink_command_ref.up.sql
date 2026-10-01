-- A downlink queued by an MQTT command carries the command's ref.
--
-- An MQTT client correlates its command with the downlink by the ref it sets;
-- downlink_queued echoes it, and every later downlink_result of the downlink
-- (sent, expired, invalid, acknowledged) carries it from this column, so the
-- client keeps no correlation state of its own. Downlinks queued by an
-- Application Center or through gRPC carry no ref and leave it NULL; rows
-- stored before this migration have none either.
--
-- The CHECK constraint holds the bound the MQTT command handler enforces
-- (storage.MaxDownlinkRefBytes): 1 to 128 bytes.
--
-- Locking: ADD COLUMN without a default takes a brief ACCESS EXCLUSIVE lock
-- and rewrites nothing; the CHECK constraint scans downlink_queue once, and
-- every existing row passes it with a NULL ref.
--
-- Idempotent: the column and the constraint are created only when missing.

ALTER TABLE downlink_queue ADD COLUMN IF NOT EXISTS ref TEXT;

DO $$
BEGIN
    IF NOT EXISTS (SELECT 1 FROM pg_constraint WHERE conname = 'downlink_queue_ref_length') THEN
        ALTER TABLE downlink_queue
            ADD CONSTRAINT downlink_queue_ref_length CHECK (ref IS NULL OR octet_length(ref) BETWEEN 1 AND 128);
    END IF;
END $$;
