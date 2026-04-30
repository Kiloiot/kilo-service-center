-- Migration 000088 DOWN: Remove multi-BS support columns

DROP INDEX IF EXISTS idx_messages_update_lookup;
DROP INDEX IF EXISTS idx_messages_duplicate;
ALTER TABLE messages DROP COLUMN IF EXISTS duplicate;
ALTER TABLE messages DROP COLUMN IF EXISTS base_stations;
