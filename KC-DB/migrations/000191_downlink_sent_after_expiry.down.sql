-- The previous release files every "sent" reported after an expiry.

ALTER TABLE downlink_queue DROP COLUMN IF EXISTS sent_after_expiry_at;
