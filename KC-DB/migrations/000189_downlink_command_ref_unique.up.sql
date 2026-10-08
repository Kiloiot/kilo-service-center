-- An MQTT command's ref names one downlink of its organization's endpoint.
--
-- A platform republishes a command/down it is not sure was received, such as
-- after a crash between its publish and its own bookkeeping. KiloCenter
-- accepts the command once: a command whose ref the organization already
-- queued for the endpoint queues nothing and reports nothing new, before and
-- after the first downlink's deadline. The unique index holds that for every
-- row, in flight or finished, so two receptions of one command racing each
-- other queue one downlink. Downlinks without a ref (Application Center,
-- gRPC) are not covered.
--
-- Backfill: refs stored since 000187 may already repeat within an endpoint
-- of an organization, one row per reception of a republished command. The
-- earliest row of each repeated ref keeps it; the later ones lose it, so
-- their results reach the client without a ref it could mistake for the
-- first command's. The upgrade preflight counts the rows that lose it.
--
-- Locking: the backfill locks the rows it updates; the index build holds a
-- SHARE lock on downlink_queue, so downlink writes wait until it is built.
--
-- Idempotent: the backfill touches only repeated refs, and the index is
-- created only when missing.

WITH ranked AS (
    SELECT id, row_number() OVER (
        PARTITION BY tenant_id, organization_id, ep_eui, ref ORDER BY id
    ) AS position
    FROM downlink_queue
    WHERE ref IS NOT NULL
)
UPDATE downlink_queue d
SET ref = NULL
FROM ranked
WHERE d.id = ranked.id AND ranked.position > 1;

CREATE UNIQUE INDEX IF NOT EXISTS uq_downlink_queue_command_ref
    ON downlink_queue (tenant_id, organization_id, ep_eui, ref)
    WHERE ref IS NOT NULL;
