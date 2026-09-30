-- The live uplink and event streams read in the order the database stored
-- the rows, by the database clock, instead of by reception or occurrence
-- time: a row stored after a newer one has streamed but carrying an older
-- time (a skewed station clock, a relayed or replayed uplink, an event whose
-- writer stamped it before committing) still reaches the open streams.
--
-- messages already carries that order in created_at, set by its NOW() default
-- and never written by the service; an index backs the stream's read.
--
-- system_events gains stored_at, set by clock_timestamp() on insert and again
-- when an UPDATE moves occurred_at forward, so the refused SCACI connect
-- that migration 000183 announces is streamed again. The column is added with
-- the constant default -infinity, which PostgreSQL records without rewriting
-- the table: the stored events keep -infinity and stay behind every stream,
-- which delivers only the rows stored after it opened. The default then
-- becomes clock_timestamp() for the rows stored from now on.
--
-- Locking: CREATE INDEX takes a SHARE lock on messages and on system_events
-- while it builds, holding back their writes (not their reads); ADD COLUMN,
-- SET DEFAULT and CREATE TRIGGER take an ACCESS EXCLUSIVE lock on
-- system_events for the catalog change only. The preflight rows "000185 ..."
-- count the rows each index covers and the writers the locks wait for.
--
-- Idempotent: the column, the indexes and the function are created only when
-- missing and the trigger is dropped before it is created.

CREATE INDEX IF NOT EXISTS idx_messages_tenant_stored
    ON messages (tenant_id, created_at DESC, id DESC)
    WHERE command_type = 'ulData';

ALTER TABLE system_events ADD COLUMN IF NOT EXISTS stored_at TIMESTAMPTZ NOT NULL DEFAULT '-infinity';
ALTER TABLE system_events ALTER COLUMN stored_at SET DEFAULT clock_timestamp();

CREATE INDEX IF NOT EXISTS idx_system_events_tenant_stored
    ON system_events (tenant_id, stored_at DESC, id DESC);

CREATE OR REPLACE FUNCTION restamp_moved_event() RETURNS trigger
    LANGUAGE plpgsql
    AS $$
BEGIN
    NEW.stored_at := clock_timestamp();
    RETURN NEW;
END;
$$;

DROP TRIGGER IF EXISTS system_events_restamp_moved ON system_events;
CREATE TRIGGER system_events_restamp_moved
    BEFORE UPDATE OF occurred_at ON system_events
    FOR EACH ROW
    WHEN (NEW.occurred_at > OLD.occurred_at)
    EXECUTE FUNCTION restamp_moved_event();
