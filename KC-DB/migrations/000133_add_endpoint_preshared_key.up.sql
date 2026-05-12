ALTER TABLE endpoints
ADD COLUMN IF NOT EXISTS preshared_key BYTEA
CHECK (preshared_key IS NULL OR length(preshared_key) = 16);

COMMENT ON COLUMN endpoints.preshared_key IS
'MIOTY preshared key for detach CMAC validation, optional per BSSCI §5.7';
