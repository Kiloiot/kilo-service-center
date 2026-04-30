-- Drop deprecated base station connectivity columns
-- host, port, tls_enabled were superseded by service_center_url
ALTER TABLE basestations DROP COLUMN IF EXISTS host;
ALTER TABLE basestations DROP COLUMN IF EXISTS port;
ALTER TABLE basestations DROP COLUMN IF EXISTS tls_enabled;
