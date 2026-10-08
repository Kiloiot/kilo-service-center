-- An endpoint's acknowledgement of a downlink is delivered through the
-- message delivery outbox.
--
-- When an uplink's dlAck acknowledges a transmitted downlink (BSSCI 3.10.1),
-- MQTT event/downlink_result publishes result "acknowledged" to the
-- organization that queued the downlink. The statement that marks the
-- downlink acknowledged now also queues one outbox row on the channel
-- mqtt_downlink_ack, naming the uplink that carried the acknowledgement
-- (message_id) and the downlink it acknowledged (acknowledged_downlink_id),
-- and the delivery worker of the core publishes it like an uplink: retried
-- while the broker is unreachable, parked on a permanent failure. A process
-- without an MQTT client, such as the federation ingress, marks the downlink
-- and the core publishes the acknowledgement.
--
-- acknowledged_downlink_id is set on exactly the mqtt_downlink_ack rows, and
-- the unique index keeps one acknowledgement row per downlink. The rows
-- stored before carry no downlink and are on the scaci or mqtt channel, so
-- none violates either. Deleting a downlink deletes its acknowledgement row.
--
-- The CHECK constraint compares the channel as text: a value added to an
-- enum cannot be used in the transaction that adds it.
--
-- Locking: ADD COLUMN without a default rewrites nothing. The foreign key,
-- the CHECK constraint and the index each scan message_delivery_outbox once
-- while the migration holds its ACCESS EXCLUSIVE lock, holding back the
-- ingest's outbox writes and the delivery worker's claims until it commits.
-- The preflight row "000188 ..." counts the rows the scans read.
--
-- Idempotent: the enum value, the column, the constraint and the index are
-- created only when missing.

ALTER TYPE message_delivery_channel ADD VALUE IF NOT EXISTS 'mqtt_downlink_ack';

ALTER TABLE message_delivery_outbox
    ADD COLUMN IF NOT EXISTS acknowledged_downlink_id BIGINT REFERENCES downlink_queue(id) ON DELETE CASCADE;

DO $$
BEGIN
    IF NOT EXISTS (SELECT 1 FROM pg_constraint WHERE conname = 'message_delivery_outbox_acknowledged_downlink') THEN
        ALTER TABLE message_delivery_outbox
            ADD CONSTRAINT message_delivery_outbox_acknowledged_downlink
            CHECK ((channel::text = 'mqtt_downlink_ack') = (acknowledged_downlink_id IS NOT NULL));
    END IF;
END $$;

CREATE UNIQUE INDEX IF NOT EXISTS uq_message_delivery_outbox_acknowledged_downlink
    ON message_delivery_outbox (acknowledged_downlink_id)
    WHERE acknowledged_downlink_id IS NOT NULL;
