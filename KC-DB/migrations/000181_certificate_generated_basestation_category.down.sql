-- The previous release files certificate.generated under audit and reads it
-- there; every such event returns to that category.
UPDATE system_events
SET event_category = 'audit'
WHERE event_type = 'certificate.generated'
  AND event_category = 'basestation';
