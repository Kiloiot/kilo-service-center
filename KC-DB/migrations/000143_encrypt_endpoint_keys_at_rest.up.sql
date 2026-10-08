-- Relax the endpoints.nwk_key / endpoints.app_key length checks so key material
-- can be stored as an authenticated encryption envelope instead of a raw
-- 16-byte value.
--
-- Key material is now written through pkg/keycrypto.Cipher (AES-256-GCM). A
-- 16-byte plaintext key becomes a 51-byte binary envelope (7-byte header +
-- 12-byte nonce + 16-byte ciphertext + 16-byte tag), so the original
-- length(nwk_key) = 16 check would reject every encrypted write. The minimum of
-- 16 bytes ensures a truncated or empty key can never be persisted, while NULL
-- remains allowed for endpoints without an application key.
--
-- Related: kilocenter-modules/pkg/keycrypto/envelope.go
--          kilocenter-modules/KC-DB/storage/postgres/endpoint_repository.go

ALTER TABLE endpoints
  DROP CONSTRAINT IF EXISTS endpoints_nwk_key_check;

ALTER TABLE endpoints
  ADD CONSTRAINT endpoints_nwk_key_check
  CHECK (nwk_key IS NULL OR length(nwk_key) >= 16);

ALTER TABLE endpoints
  DROP CONSTRAINT IF EXISTS endpoints_app_key_check;

ALTER TABLE endpoints
  ADD CONSTRAINT endpoints_app_key_check
  CHECK (app_key IS NULL OR length(app_key) >= 16);

COMMENT ON COLUMN endpoints.nwk_key IS 'Network session key (nwkSnKey) stored as an AES-256-GCM envelope via pkg/keycrypto (~51 bytes). Decrypted to the 16-byte wire value on read.';
COMMENT ON COLUMN endpoints.app_key IS 'Application session key (appKey) stored as an AES-256-GCM envelope via pkg/keycrypto (~51 bytes), or NULL when unprovisioned.';
