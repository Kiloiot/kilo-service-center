-- An unhandled system event is 'new': the state the alert API reports and
-- filters by (models.EventStatusNew). Events were stored with the column
-- default 'active' because the insert never wrote the status.
--
-- system_events can be large, so the file runs as short transactions (each
-- COMMIT ends one) and no row scan runs under ACCESS EXCLUSIVE: the catalog
-- changes lock the table only for an instant, the data change takes row
-- locks, and the validations take SHARE UPDATE EXCLUSIVE, which reads and
-- writes pass. Every step can be rerun after a failure.
-- idx_system_events_status (status IN ('new', 'active')) keeps serving the
-- 'new' rows unchanged, so no index is rebuilt.

-- New rows are 'new' from here on.
ALTER TABLE system_events ALTER COLUMN status SET DEFAULT 'new';
COMMIT;

UPDATE system_events SET status = 'new' WHERE status IS NULL OR status = 'active';
COMMIT;

-- The constraints hold for rows written from here on; existing rows are
-- checked below.
ALTER TABLE system_events
    DROP CONSTRAINT IF EXISTS system_events_status_check,
    DROP CONSTRAINT IF EXISTS system_events_status_present,
    ADD CONSTRAINT system_events_status_check
        CHECK (status IN ('new', 'acknowledged', 'resolved', 'ignored')) NOT VALID,
    ADD CONSTRAINT system_events_status_present
        CHECK (status IS NOT NULL) NOT VALID;
COMMIT;

ALTER TABLE system_events VALIDATE CONSTRAINT system_events_status_check;
ALTER TABLE system_events VALIDATE CONSTRAINT system_events_status_present;
COMMIT;

-- SET NOT NULL proves itself from the validated check instead of scanning.
ALTER TABLE system_events ALTER COLUMN status SET NOT NULL;
ALTER TABLE system_events DROP CONSTRAINT system_events_status_present;
