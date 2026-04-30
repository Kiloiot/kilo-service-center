-- Rollback migration 000051

DROP INDEX IF EXISTS idx_basestations_bidi;

ALTER TABLE basestation_sessions
DROP COLUMN IF EXISTS bidi;

ALTER TABLE basestations
DROP COLUMN IF EXISTS bidi;