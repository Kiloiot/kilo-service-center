-- Drop the packet counter reuse flag; reused counters are delivered as new data again.

ALTER TABLE messages_archive DROP COLUMN IF EXISTS packet_cnt_reused;
ALTER TABLE messages DROP COLUMN IF EXISTS packet_cnt_reused;

COMMENT ON COLUMN messages.duplicate IS 'True if packet counter was reused (duplicate detection) per SCACI §3.8.1';
