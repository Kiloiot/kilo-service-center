-- Rollback migration 072: Remove nwk_sn_key column
ALTER TABLE messages DROP COLUMN IF EXISTS nwk_sn_key;
