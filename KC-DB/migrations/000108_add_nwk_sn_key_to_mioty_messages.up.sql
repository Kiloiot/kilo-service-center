-- Migration 072: Add encrypted network session key for attach propagate messages
-- BSSCI §5.8.1 + §5.6.2: Encrypted key storage for propagate audit trail
-- Other propagate fields (bidi, last_packet_cnt) stored in metadata JSONB
--
-- Note: Migration 047 renamed mioty_messages → messages. This migration targets the correct table.

ALTER TABLE messages ADD COLUMN IF NOT EXISTS nwk_sn_key BYTEA;

COMMENT ON COLUMN messages.nwk_sn_key IS 'Encrypted network session key (16 bytes, BSSCI §5.6.2)';
