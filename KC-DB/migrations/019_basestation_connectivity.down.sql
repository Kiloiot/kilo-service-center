-- Rollback migration: Remove MIOTY-compliant Base Station connectivity fields

-- Drop audit event table
DROP TABLE IF EXISTS basestation_connection_events;

-- Drop certificate table
DROP TABLE IF EXISTS basestation_certificates;

-- Drop indexes
DROP INDEX IF EXISTS idx_basestations_connection_type;
DROP INDEX IF EXISTS idx_basestations_service_center;
DROP INDEX IF EXISTS idx_basestations_online_status;
DROP INDEX IF EXISTS idx_basestations_retry;

-- Drop constraints
ALTER TABLE basestations
DROP CONSTRAINT IF EXISTS check_bssci_config,
DROP CONSTRAINT IF EXISTS check_mqtt_config;

-- Remove columns from basestations table
ALTER TABLE basestations
DROP COLUMN IF EXISTS connection_type,
DROP COLUMN IF EXISTS service_center_url,
DROP COLUMN IF EXISTS mqtt_broker_url,
DROP COLUMN IF EXISTS mqtt_client_id,
DROP COLUMN IF EXISTS mqtt_username,
DROP COLUMN IF EXISTS mqtt_password_encrypted,
DROP COLUMN IF EXISTS mqtt_topic_prefix,
DROP COLUMN IF EXISTS mqtt_topics,
DROP COLUMN IF EXISTS tls_ca_certificate,
DROP COLUMN IF EXISTS tls_certificate,
DROP COLUMN IF EXISTS tls_key,
DROP COLUMN IF EXISTS tls_auth_required,
DROP COLUMN IF EXISTS tls_hostname_verification,
DROP COLUMN IF EXISTS tls_cert_fingerprint,
DROP COLUMN IF EXISTS tls_cert_expires_at,
DROP COLUMN IF EXISTS connection_status,
DROP COLUMN IF EXISTS last_error,
DROP COLUMN IF EXISTS retry_count,
DROP COLUMN IF EXISTS next_retry_at,
DROP COLUMN IF EXISTS config_file_content,
DROP COLUMN IF EXISTS config_file_uploaded_at;

-- Drop connection_type enum
DROP TYPE IF EXISTS connection_type;

-- Remove deprecation comment from host field
COMMENT ON COLUMN basestations.host IS NULL;
