-- Migration: Add MIOTY-compliant Base Station connectivity fields
-- Supports both BSSCI (standard) and MQTT (alternative) connection methods
-- Enables location-independent Base Station connections

-- Add connection type enum
CREATE TYPE connection_type AS ENUM ('bssci', 'mqtt');

-- Add connection type and configuration fields
ALTER TABLE basestations
ADD COLUMN connection_type connection_type DEFAULT 'bssci',
ADD COLUMN service_center_url TEXT,
ADD COLUMN mqtt_broker_url TEXT,
ADD COLUMN mqtt_client_id VARCHAR(255),
ADD COLUMN mqtt_username VARCHAR(255),
ADD COLUMN mqtt_password_encrypted BYTEA,
ADD COLUMN mqtt_topic_prefix VARCHAR(255),
ADD COLUMN tls_ca_certificate TEXT,
ADD COLUMN tls_certificate TEXT,
ADD COLUMN tls_key TEXT,
ADD COLUMN tls_auth_required BOOLEAN DEFAULT true,
ADD COLUMN tls_hostname_verification BOOLEAN DEFAULT true,
ADD COLUMN tls_cert_fingerprint VARCHAR(64),
ADD COLUMN tls_cert_expires_at TIMESTAMP WITH TIME ZONE,
ADD COLUMN connection_status JSONB DEFAULT '{}',
ADD COLUMN last_error TEXT,
ADD COLUMN retry_count INTEGER DEFAULT 0,
ADD COLUMN next_retry_at TIMESTAMP WITH TIME ZONE,
ADD COLUMN config_file_content TEXT,
ADD COLUMN config_file_uploaded_at TIMESTAMP WITH TIME ZONE;

-- Mark old host field as deprecated
COMMENT ON COLUMN basestations.host IS 'DEPRECATED - use service_center_url instead. Will be removed in future version.';

-- Add comments for new fields
COMMENT ON COLUMN basestations.connection_type IS 'Connection protocol: bssci (Base Station Service Center Interface) or mqtt';
COMMENT ON COLUMN basestations.service_center_url IS 'Full URL of the Service Center (e.g., https://sc.example.com:8443)';
COMMENT ON COLUMN basestations.mqtt_broker_url IS 'MQTT broker URL for MQTT-based connections';
COMMENT ON COLUMN basestations.mqtt_client_id IS 'MQTT client ID, typically derived from Base Station EUI';
COMMENT ON COLUMN basestations.mqtt_username IS 'MQTT authentication username';
COMMENT ON COLUMN basestations.mqtt_password_encrypted IS 'Encrypted MQTT password';
COMMENT ON COLUMN basestations.mqtt_topic_prefix IS 'MQTT topic prefix for this Base Station';
COMMENT ON COLUMN basestations.tls_ca_certificate IS 'TLS CA certificate in PEM format';
COMMENT ON COLUMN basestations.tls_certificate IS 'TLS client certificate in PEM format';
COMMENT ON COLUMN basestations.tls_key IS 'TLS client private key in PEM format (encrypted)';
COMMENT ON COLUMN basestations.tls_auth_required IS 'Whether mutual TLS authentication is required';
COMMENT ON COLUMN basestations.tls_hostname_verification IS 'Whether to verify Service Center hostname in TLS handshake';
COMMENT ON COLUMN basestations.tls_cert_fingerprint IS 'SHA256 fingerprint of TLS certificate';
COMMENT ON COLUMN basestations.tls_cert_expires_at IS 'TLS certificate expiration timestamp';
COMMENT ON COLUMN basestations.connection_status IS 'Detailed connection state and metrics';
COMMENT ON COLUMN basestations.last_error IS 'Last connection error for troubleshooting';
COMMENT ON COLUMN basestations.retry_count IS 'Number of connection retry attempts';
COMMENT ON COLUMN basestations.next_retry_at IS 'Next retry timestamp for exponential backoff';
COMMENT ON COLUMN basestations.config_file_content IS 'Original configuration file content if uploaded';
COMMENT ON COLUMN basestations.config_file_uploaded_at IS 'When configuration file was uploaded';

-- Add indexes for performance
CREATE INDEX idx_basestations_connection_type ON basestations(connection_type);
CREATE INDEX idx_basestations_service_center ON basestations(service_center_url) WHERE service_center_url IS NOT NULL;
CREATE INDEX idx_basestations_online_status ON basestations(is_online, last_seen_at);
CREATE INDEX idx_basestations_retry ON basestations(next_retry_at) 
    WHERE next_retry_at IS NOT NULL AND is_online = false;

-- Add constraints
ALTER TABLE basestations
ADD CONSTRAINT check_bssci_config CHECK (
    connection_type != 'bssci' OR (
        service_center_url IS NOT NULL AND
        tls_ca_certificate IS NOT NULL
    )
),
ADD CONSTRAINT check_mqtt_config CHECK (
    connection_type != 'mqtt' OR (
        mqtt_broker_url IS NOT NULL
    )
);

-- Create table for storing Base Station certificates (for BSSCI)
CREATE TABLE IF NOT EXISTS basestation_certificates (
    id BIGSERIAL PRIMARY KEY,
    basestation_id BIGINT NOT NULL REFERENCES basestations(id) ON DELETE CASCADE,
    certificate_type VARCHAR(20) NOT NULL CHECK (certificate_type IN ('client', 'ca')),
    certificate_pem TEXT NOT NULL,
    private_key_encrypted BYTEA, -- Encrypted private key for client certs
    fingerprint VARCHAR(64) NOT NULL,
    subject_dn TEXT NOT NULL,
    issuer_dn TEXT NOT NULL,
    not_before TIMESTAMP WITH TIME ZONE NOT NULL,
    not_after TIMESTAMP WITH TIME ZONE NOT NULL,
    is_active BOOLEAN DEFAULT true,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT unique_active_cert_per_type UNIQUE(basestation_id, certificate_type, is_active)
);

-- Add index for certificate lookups
CREATE INDEX idx_basestation_certificates_active ON basestation_certificates(basestation_id, is_active)
    WHERE is_active = true;
CREATE INDEX idx_basestation_certificates_expiry ON basestation_certificates(not_after)
    WHERE is_active = true;

-- Add comments for certificate table
COMMENT ON TABLE basestation_certificates IS 'TLS certificates for BSSCI Base Station connections';
COMMENT ON COLUMN basestation_certificates.certificate_type IS 'Type: client (Base Station cert) or ca (Service Center CA)';
COMMENT ON COLUMN basestation_certificates.certificate_pem IS 'X.509 certificate in PEM format';
COMMENT ON COLUMN basestation_certificates.private_key_encrypted IS 'Encrypted private key for client certificates';
COMMENT ON COLUMN basestation_certificates.fingerprint IS 'SHA256 fingerprint of certificate';
COMMENT ON COLUMN basestation_certificates.subject_dn IS 'Certificate subject distinguished name';
COMMENT ON COLUMN basestation_certificates.issuer_dn IS 'Certificate issuer distinguished name';

-- Create audit table for connection events
CREATE TABLE IF NOT EXISTS basestation_connection_events (
    id BIGSERIAL PRIMARY KEY,
    basestation_id BIGINT NOT NULL REFERENCES basestations(id) ON DELETE CASCADE,
    event_type VARCHAR(50) NOT NULL,
    event_data JSONB DEFAULT '{}',
    remote_address INET,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

-- Add index for event queries
CREATE INDEX idx_basestation_connection_events_bs ON basestation_connection_events(basestation_id, created_at DESC);
CREATE INDEX idx_basestation_connection_events_type ON basestation_connection_events(event_type, created_at DESC);

-- Add comments for event table
COMMENT ON TABLE basestation_connection_events IS 'Audit log of Base Station connection events';
COMMENT ON COLUMN basestation_connection_events.event_type IS 'Event type: connected, disconnected, auth_failed, etc.';
COMMENT ON COLUMN basestation_connection_events.event_data IS 'Additional event-specific data';
COMMENT ON COLUMN basestation_connection_events.remote_address IS 'Remote IP address of Base Station';

-- Update existing Base Stations to have default connection type
UPDATE basestations 
SET connection_type = 'bssci' 
WHERE connection_type IS NULL;
