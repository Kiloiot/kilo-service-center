-- Hands unhandled events back as 'active' in the same short transactions as
-- the upgrade: no row scan runs under ACCESS EXCLUSIVE.

ALTER TABLE system_events
    ALTER COLUMN status DROP NOT NULL,
    ALTER COLUMN status SET DEFAULT 'active',
    DROP CONSTRAINT IF EXISTS system_events_status_check,
    ADD CONSTRAINT system_events_status_check
        CHECK (status IN ('new', 'active', 'acknowledged', 'resolved', 'ignored')) NOT VALID;
COMMIT;

UPDATE system_events SET status = 'active' WHERE status = 'new';
COMMIT;

ALTER TABLE system_events VALIDATE CONSTRAINT system_events_status_check;
