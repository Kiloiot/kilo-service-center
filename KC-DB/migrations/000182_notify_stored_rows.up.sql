-- Every stored uplink and every stored system event is announced on a
-- notification channel, so the uplink and event streams read it at once
-- instead of on their next poll. Every writer is covered: KC-Core, KC-Identity
-- and any further KC-Core sharing the database. The payload is empty: a
-- stream re-reads through its own tenant-scoped query, so a notification
-- carries no data of any tenant. The channel names are the
-- postgres.ChannelUplinkStored and postgres.ChannelEventStored constants.
--
-- Only INSERT announces: the streams deliver the rows stored after they
-- opened, and a later UPDATE of a row is never streamed again.
--
-- Locking: CREATE TRIGGER takes a SHARE ROW EXCLUSIVE lock on messages and on
-- system_events. It waits for the open transactions writing them and blocks
-- their writes (not their reads) for the catalog change only; no row is read
-- or rewritten. The preflight row "000182 open writes on the announced
-- tables" counts the writers it would wait for.
--
-- Idempotent: the function is replaced and each trigger dropped before it is
-- created.

CREATE OR REPLACE FUNCTION notify_row_stored() RETURNS trigger
    LANGUAGE plpgsql
    AS $$
BEGIN
    PERFORM pg_notify(TG_ARGV[0], '');
    RETURN NULL;
END;
$$;

DROP TRIGGER IF EXISTS messages_notify_uplink_stored ON messages;
CREATE TRIGGER messages_notify_uplink_stored
    AFTER INSERT ON messages
    FOR EACH ROW
    WHEN (NEW.command_type = 'ulData')
    EXECUTE FUNCTION notify_row_stored('kc_uplink_stored');

DROP TRIGGER IF EXISTS system_events_notify_event_stored ON system_events;
CREATE TRIGGER system_events_notify_event_stored
    AFTER INSERT ON system_events
    FOR EACH ROW
    EXECUTE FUNCTION notify_row_stored('kc_event_stored');
