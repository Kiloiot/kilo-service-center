-- Create dedicated sequence for que_id (separate from id BIGSERIAL)
-- que_id is the BSSCI operation ID, distinct from the table primary key
CREATE SEQUENCE IF NOT EXISTS downlink_queue_que_id_seq;

-- Backfill: set sequence to MAX(que_id)+1 to avoid collisions with existing rows
-- setval(seq, value, false) means next nextval() returns value (not value+1)
SELECT setval('downlink_queue_que_id_seq',
    GREATEST(COALESCE((SELECT MAX(que_id) FROM downlink_queue), 0), 0) + 1,
    false);

-- Set column default so inserts don't need explicit nextval
ALTER TABLE downlink_queue
  ALTER COLUMN que_id SET DEFAULT nextval('downlink_queue_que_id_seq');

-- Comment for schema introspection
COMMENT ON SEQUENCE downlink_queue_que_id_seq IS 'Dedicated sequence for que_id (BSSCI operation ID, distinct from PK)';
