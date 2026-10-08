-- A base station certificate is issued by the station's managers, and its
-- certificate.generated event is filed under the basestation category they
-- read. Events stored before that were filed under audit, which only
-- administrators read; this moves them to basestation. The server
-- certificate events stay under audit.
--
-- Locking: one UPDATE that matches only the audit rows of that event type
-- (found through idx_system_events_category). It takes the ROW EXCLUSIVE
-- table lock every UPDATE takes, which blocks no reads, inserts or updates
-- of other rows, and row locks on the rows it rewrites. Their number is the
-- preflight row "000181 certificate.generated events filed under audit".
--
-- Idempotent: a moved row is no longer under audit and no longer matches.

UPDATE system_events
SET event_category = 'basestation'
WHERE event_type = 'certificate.generated'
  AND event_category = 'audit';
