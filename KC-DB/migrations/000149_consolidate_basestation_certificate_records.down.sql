-- Recreate the legacy basestation_certificates table as migration 019 defined
-- it. It comes back empty: the canonical basestations.tls_* columns keep the
-- certificates the up migration consolidated, and the archived metadata in
-- system_events never held PEM bodies or key material to restore from.

CREATE TABLE IF NOT EXISTS basestation_certificates (
    id BIGSERIAL PRIMARY KEY,
    basestation_id BIGINT NOT NULL REFERENCES basestations(id) ON DELETE CASCADE,
    certificate_type VARCHAR(20) NOT NULL CHECK (certificate_type IN ('client', 'ca')),
    certificate_pem TEXT NOT NULL,
    private_key_encrypted BYTEA,
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

CREATE INDEX IF NOT EXISTS idx_basestation_certificates_active ON basestation_certificates(basestation_id, is_active)
    WHERE is_active = true;
CREATE INDEX IF NOT EXISTS idx_basestation_certificates_expiry ON basestation_certificates(not_after)
    WHERE is_active = true;

COMMENT ON TABLE basestation_certificates IS 'TLS certificates for BSSCI Base Station connections';
