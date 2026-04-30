-- Migration 100 down: Restore original event category constraint
-- WARNING: This will fail if any rows have category values from the extended set

ALTER TABLE system_events
DROP CONSTRAINT IF EXISTS system_events_event_category_check;

ALTER TABLE system_events
ADD CONSTRAINT system_events_event_category_check
CHECK (event_category IN (
    'security', 'endpoint', 'basestation', 'message', 'system', 'roaming', 'error', 'audit'
));

COMMENT ON COLUMN system_events.event_category IS 'High-level event category for filtering';
