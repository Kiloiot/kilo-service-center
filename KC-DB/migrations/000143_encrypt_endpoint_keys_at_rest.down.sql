-- Restore the strict 16-byte length checks on endpoints.nwk_key / app_key.
-- Rolling back requires that every stored key has already been decrypted back to
-- its raw 16-byte form; encrypted envelopes (~51 bytes) will violate these
-- constraints.

ALTER TABLE endpoints
  DROP CONSTRAINT IF EXISTS endpoints_nwk_key_check;

ALTER TABLE endpoints
  ADD CONSTRAINT endpoints_nwk_key_check
  CHECK (length(nwk_key) = 16);

ALTER TABLE endpoints
  DROP CONSTRAINT IF EXISTS endpoints_app_key_check;

ALTER TABLE endpoints
  ADD CONSTRAINT endpoints_app_key_check
  CHECK (length(app_key) = 16);

COMMENT ON COLUMN endpoints.nwk_key IS 'Network Key';
COMMENT ON COLUMN endpoints.app_key IS 'Application Key';
