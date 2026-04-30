-- Remove MIOTY radio metrics and attach fields

-- Remove radio metrics fields
ALTER TABLE endpoints
DROP COLUMN IF EXISTS last_snr,
DROP COLUMN IF EXISTS last_rssi,
DROP COLUMN IF EXISTS last_eq_snr,
DROP COLUMN IF EXISTS last_profile;

-- Remove attach fields  
ALTER TABLE endpoints
DROP COLUMN IF EXISTS attach_cnt,
DROP COLUMN IF EXISTS nonce,
DROP COLUMN IF EXISTS sign,
DROP COLUMN IF EXISTS last_attach_rx_time,
DROP COLUMN IF EXISTS last_attach_rx_duration;

-- Remove indexes
DROP INDEX IF EXISTS idx_endpoints_last_snr;
DROP INDEX IF EXISTS idx_endpoints_last_rssi;
DROP INDEX IF EXISTS idx_endpoints_attach_cnt;