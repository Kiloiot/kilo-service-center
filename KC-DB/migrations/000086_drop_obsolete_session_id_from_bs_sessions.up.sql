-- Drop obsolete session_id column from basestation_sessions
-- This column was added in migration 007 but was superseded by sn_bs_uuid and sn_sc_uuid in migration 028.
-- The session_id column is NOT NULL without a default, but no code path ever populates it,
-- causing CreateSession to fail with "null value in column session_id violates not-null constraint".
--
-- The session UUIDs are now tracked via:
-- - sn_bs_uuid: Base Station session UUID (for resumption matching)
-- - sn_sc_uuid: Service Center session UUID (for resumption matching)
--
-- These provide the same functionality as the original session_id, making it redundant.

-- First, drop the index on session_id (if it exists)
DROP INDEX IF EXISTS idx_basestation_sessions_session;

-- Then drop the column
ALTER TABLE basestation_sessions DROP COLUMN IF EXISTS session_id;
