-- A downlink an Application Center queued carries the EUI of that Application
-- Center.
--
-- SCACI §3.12 reports a downlink's result to the Application Center that
-- queued it. The queue row kept only the id that Application Center assigned
-- (ac_que_id); its identity lived in the operation log alone, whose write
-- happens after the row is committed and whose failure did not refuse the
-- downlink. A lost record lost the result's destination for good. ac_eui is
-- written with the row by the dlDataQue that queues it; downlinks enqueued
-- without an Application Center (gRPC, MQTT) leave it NULL. The organization
-- and tenant of the Application Center are the row's own.
--
-- Backfill: each row that an Application Center queue id names takes the
-- acEui of the session whose inbound dlDataQue record names the row, inside
-- the row's tenant, organization and endpoint. A record carrying scQueId
-- names the row by its service center queue id; an older record carries only
-- queId, the Application Center's id (migration 000160 copied que_id, the id
-- such a record carried, into ac_que_id). The scQueId evidence wins over the
-- older one. A row whose evidence names no Application Center, or more than
-- one, stays NULL: its result reaches no Application Center rather than a
-- guessed one. The upgrade preflight lists the in-flight rows left so.
--
-- Locking: ADD COLUMN takes a brief ACCESS EXCLUSIVE lock; the backfill locks
-- the rows it updates; the two CHECK constraints scan downlink_queue once.
--
-- Idempotent: the column and constraints are created only when missing, and
-- the backfill touches only rows without an origin.

ALTER TABLE downlink_queue ADD COLUMN IF NOT EXISTS ac_eui BYTEA;

WITH evidence AS (
    SELECT d.id AS downlink_id, s.ac_eui,
           CASE WHEN o.request_data ? 'scQueId' THEN 1 ELSE 2 END AS strength
    FROM downlink_queue d
    JOIN scaci_operation_log o
      ON o.tenant_id = d.tenant_id AND o.command = 'dlDataQue' AND o.direction = 'inbound'
    JOIN scaci_sessions s
      ON s.id = o.session_id AND s.tenant_id = o.tenant_id
     AND s.organization_id IS NOT DISTINCT FROM d.organization_id
    WHERE d.ac_eui IS NULL AND d.ac_que_id IS NOT NULL
      AND octet_length(s.ac_eui) = 8
      AND upper(o.request_data->>'epEui') = upper(encode(d.ep_eui, 'hex'))
      AND (o.request_data->>'scQueId' = d.que_id::text
           OR (NOT (o.request_data ? 'scQueId') AND o.request_data->>'queId' = d.ac_que_id::text))
),
strongest AS (
    SELECT downlink_id, min(strength) AS strength FROM evidence GROUP BY downlink_id
),
resolved AS (
    SELECT e.downlink_id, (array_agg(e.ac_eui))[1] AS ac_eui
    FROM evidence e
    JOIN strongest USING (downlink_id)
    WHERE e.strength = strongest.strength
    GROUP BY e.downlink_id
    HAVING count(DISTINCT e.ac_eui) = 1
)
UPDATE downlink_queue d
SET ac_eui = resolved.ac_eui
FROM resolved
WHERE d.id = resolved.downlink_id;

DO $$
BEGIN
    IF NOT EXISTS (SELECT 1 FROM pg_constraint WHERE conname = 'downlink_queue_ac_eui_length') THEN
        ALTER TABLE downlink_queue
            ADD CONSTRAINT downlink_queue_ac_eui_length CHECK (ac_eui IS NULL OR octet_length(ac_eui) = 8);
    END IF;
    IF NOT EXISTS (SELECT 1 FROM pg_constraint WHERE conname = 'downlink_queue_ac_eui_queue_id') THEN
        ALTER TABLE downlink_queue
            ADD CONSTRAINT downlink_queue_ac_eui_queue_id CHECK (ac_eui IS NULL OR ac_que_id IS NOT NULL);
    END IF;
END $$;
