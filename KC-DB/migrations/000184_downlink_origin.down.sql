-- The queue row no longer carries the Application Center that queued it; the
-- previous release names it from the operation log.

ALTER TABLE downlink_queue DROP CONSTRAINT IF EXISTS downlink_queue_ac_eui_queue_id;
ALTER TABLE downlink_queue DROP CONSTRAINT IF EXISTS downlink_queue_ac_eui_length;
ALTER TABLE downlink_queue DROP COLUMN IF EXISTS ac_eui;
