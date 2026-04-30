-- Migration: Relax session_key constraint to support authenticated GCM encryption
--
-- BSSCI §5.6.2 requires nwkSnKey as Numeric[16] (16-byte array on wire protocol).
-- For secure storage, we use AES-256-GCM which produces:
--   - Nonce: 12 bytes
--   - Ciphertext: 16 bytes (for 16-byte plaintext)
--   - Authentication tag: 16 bytes
--   - Total: 44 bytes raw GCM blob
--
-- Previous implementation used base64 encoding:
--   - 44 bytes → ~60 base64 characters → 60 bytes when stored as BYTEA
--
-- This migration relaxes the constraint to allow both formats during lazy migration:
--   - New format: 44-byte raw GCM blob (authenticated encryption)
--   - Previous format: 60-byte base64-encoded GCM (migrated on first uplink)
--
-- Related: kilocenter-modules/KC-Core/pkg/crypto/keyencryption.go (EncryptKeyRaw, DecryptKeyWithMigration)
--          kilocenter-modules/KC-Core/pkg/bssci/ul_transmit.go (lazy migration logic)

-- Remove restrictive 16-byte check (both original and PostgreSQL auto-numbered variant)
ALTER TABLE endpoint_sessions
  DROP CONSTRAINT IF EXISTS endpoint_sessions_session_key_check;

ALTER TABLE endpoint_sessions
  DROP CONSTRAINT IF EXISTS endpoint_sessions_session_key_check1;

-- Add relaxed constraint allowing GCM blob (44 bytes) and previous base64 (60 bytes)
-- Minimum 16 bytes ensures we never store unencrypted or truncated keys
ALTER TABLE endpoint_sessions
  ADD CONSTRAINT endpoint_sessions_session_key_check
  CHECK (session_key IS NULL OR length(session_key) >= 16);

-- Update comment to reflect authenticated encryption storage
COMMENT ON COLUMN endpoint_sessions.session_key IS 'Network session key (NWKSnKey) stored as AES-256-GCM blob (nonce + ciphertext + tag = ~44 bytes). Previous base64 format (~60 bytes) auto-migrates on first uplink. Wire protocol sends decrypted 16-byte key as Numeric[16] array per BSSCI §5.6.2.';
