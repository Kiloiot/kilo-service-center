-- Stored uplinks and system events are no longer announced; the previous
-- release reads them on its poll alone.

DROP TRIGGER IF EXISTS system_events_notify_event_stored ON system_events;
DROP TRIGGER IF EXISTS messages_notify_uplink_stored ON messages;
DROP FUNCTION IF EXISTS notify_row_stored();
