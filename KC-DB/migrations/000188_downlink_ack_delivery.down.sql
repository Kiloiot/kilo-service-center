-- The outbox no longer carries endpoint acknowledgements; the previous release
-- publishes them from the process that marks the downlink. The rows still
-- queued are deleted with the column. The enum value mqtt_downlink_ack stays:
-- PostgreSQL cannot remove a value from an enum, and no row uses it.

DELETE FROM message_delivery_outbox WHERE acknowledged_downlink_id IS NOT NULL;
DROP INDEX IF EXISTS uq_message_delivery_outbox_acknowledged_downlink;
ALTER TABLE message_delivery_outbox DROP CONSTRAINT IF EXISTS message_delivery_outbox_acknowledged_downlink;
ALTER TABLE message_delivery_outbox DROP COLUMN IF EXISTS acknowledged_downlink_id;
