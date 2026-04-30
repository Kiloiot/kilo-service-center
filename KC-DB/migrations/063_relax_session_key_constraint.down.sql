-- Migration rollback: Restore strict 16-byte session_key constraint
--
-- WARNING: This rollback will FAIL if any session keys are still in GCM blob format (44 bytes)
--          or previous base64 format (60 bytes). Only run this after ALL sessions have been
--          migrated to a hypothetical future 16-byte-only encryption scheme.
--
-- Current production state likely has:
--   - New format: 44-byte GCM blobs
--   - Previous format: 60-byte base64 GCM
--
-- This down migration is provided for schema consistency but should NOT be run
-- unless you've implemented and migrated to a different encryption approach.

-- Remove relaxed constraint
ALTER TABLE endpoint_sessions
  DROP CONSTRAINT IF EXISTS endpoint_sessions_session_key_check;

-- Restore strict 16-byte constraint (original from migration 015)
-- This will FAIL if any keys are not exactly 16 bytes
ALTER TABLE endpoint_sessions
  ADD CONSTRAINT endpoint_sessions_session_key_check
  CHECK (session_key IS NULL OR length(session_key) = 16);

-- Restore original comment
COMMENT ON COLUMN endpoint_sessions.session_key IS 'Network session key (NWKSnKey) derived from network key for this session';
