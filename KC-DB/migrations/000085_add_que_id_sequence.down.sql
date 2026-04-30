-- Remove que_id default and drop dedicated sequence
ALTER TABLE downlink_queue ALTER COLUMN que_id DROP DEFAULT;
DROP SEQUENCE IF EXISTS downlink_queue_que_id_seq;
