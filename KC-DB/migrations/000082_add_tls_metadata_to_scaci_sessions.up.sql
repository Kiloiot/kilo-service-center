-- Migration 000082: Add TLS metadata columns to scaci_sessions
-- Per SCACI §1 TLS evidence persistence requirements

ALTER TABLE scaci_sessions
ADD COLUMN tls_version VARCHAR(16),
ADD COLUMN cipher_suite VARCHAR(64);

-- Index for filtering by TLS version within a tenant (compliance reporting)
CREATE INDEX idx_scaci_sessions_tenant_tls ON scaci_sessions(tenant_id, tls_version);

COMMENT ON COLUMN scaci_sessions.tls_version IS 'TLS version negotiated at connect (e.g., TLS 1.2, TLS 1.3)';
COMMENT ON COLUMN scaci_sessions.cipher_suite IS 'TLS cipher suite negotiated at connect';
