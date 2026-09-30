-- A system event moved forward in time is no longer announced; the previous
-- release streams it again on its poll alone.

DROP TRIGGER IF EXISTS system_events_notify_event_moved ON system_events;
