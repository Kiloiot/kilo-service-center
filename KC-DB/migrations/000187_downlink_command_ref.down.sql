-- The queue row no longer carries the ref of the MQTT command that queued it;
-- the previous release echoes the ref in downlink_queued only.

ALTER TABLE downlink_queue DROP CONSTRAINT IF EXISTS downlink_queue_ref_length;
ALTER TABLE downlink_queue DROP COLUMN IF EXISTS ref;
