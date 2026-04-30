-- Migration 000082 down: Remove TLS metadata columns from scaci_sessions

DROP INDEX IF EXISTS idx_scaci_sessions_tenant_tls;
ALTER TABLE scaci_sessions DROP COLUMN IF EXISTS cipher_suite;
ALTER TABLE scaci_sessions DROP COLUMN IF EXISTS tls_version;
