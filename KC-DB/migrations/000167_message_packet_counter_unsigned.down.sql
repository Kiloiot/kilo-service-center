-- Return packet_cnt to INTEGER. The guard aborts while a stored or archived
-- message holds a packet counter at or above 2^31, which INTEGER cannot store.

DO $$
DECLARE
    beyond bigint;
BEGIN
    SELECT (SELECT count(*) FROM messages WHERE packet_cnt > 2147483647)
         + (SELECT count(*) FROM messages_archive WHERE packet_cnt > 2147483647)
      INTO beyond;
    IF beyond > 0 THEN RAISE EXCEPTION 'packet counter exceeds INTEGER on % row(s)', beyond; END IF;
END $$;

ALTER TABLE messages DROP CONSTRAINT valid_packet_cnt;
ALTER TABLE messages ALTER COLUMN packet_cnt TYPE INTEGER;
ALTER TABLE messages ADD CONSTRAINT valid_packet_cnt CHECK (packet_cnt >= 0);

ALTER TABLE messages_archive DROP CONSTRAINT messages_archive_valid_packet_cnt;
ALTER TABLE messages_archive ALTER COLUMN packet_cnt TYPE INTEGER;
ALTER TABLE messages_archive ADD CONSTRAINT messages_archive_valid_packet_cnt CHECK (packet_cnt >= 0);
