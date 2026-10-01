-- A system event moved forward in time is announced on the event channel as
-- a stored one is. The event stream delivers every row whose occurred_at is
-- past the newest one it delivered, so a row an UPDATE moves forward is
-- streamed again on the stream's next read; without the announcement that
-- read waits for the stream's poll. The refused SCACI connect is such a row:
-- a recurrence of an open refusal moves its record to the new time and counts
-- it instead of inserting a record. An UPDATE that leaves occurred_at where
-- it was, or moves it back, is never streamed again and announces nothing.
--
-- The trigger reuses notify_row_stored() of migration 000182 and its channel,
-- postgres.ChannelEventStored; the payload stays empty.
--
-- Locking: CREATE TRIGGER takes a SHARE ROW EXCLUSIVE lock on system_events.
-- It waits for the open transactions writing it and blocks their writes (not
-- their reads) for the catalog change only; no row is read or rewritten. The
-- preflight row "000183 open writes on system_events" counts the writers it
-- would wait for.
--
-- Idempotent: the trigger is dropped before it is created.

DROP TRIGGER IF EXISTS system_events_notify_event_moved ON system_events;
CREATE TRIGGER system_events_notify_event_moved
    AFTER UPDATE OF occurred_at ON system_events
    FOR EACH ROW
    WHEN (NEW.occurred_at > OLD.occurred_at)
    EXECUTE FUNCTION notify_row_stored('kc_event_stored');
