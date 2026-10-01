-- Hold the endpoint packet counter over its full unsigned 32-bit range.
--
-- BSSCI §3.10.1 and radio §3.6.5.3 define packetCnt as a 32-bit counter. The
-- INTEGER columns refused every uplink from 2^31 on. BIGINT holds every value
-- and the checks bound it to the unsigned 32-bit range. The type change
-- rewrites both tables and the indexes that include packet_cnt.

ALTER TABLE messages DROP CONSTRAINT valid_packet_cnt;
ALTER TABLE messages ALTER COLUMN packet_cnt TYPE BIGINT;
ALTER TABLE messages
    ADD CONSTRAINT valid_packet_cnt CHECK (packet_cnt >= 0 AND packet_cnt <= 4294967295);

ALTER TABLE messages_archive DROP CONSTRAINT messages_archive_valid_packet_cnt;
ALTER TABLE messages_archive ALTER COLUMN packet_cnt TYPE BIGINT;
ALTER TABLE messages_archive
    ADD CONSTRAINT messages_archive_valid_packet_cnt CHECK (packet_cnt >= 0 AND packet_cnt <= 4294967295);
