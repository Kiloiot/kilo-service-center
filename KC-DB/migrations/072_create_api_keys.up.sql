-- Migration 072: Create api_keys table for programmatic access authentication
--
-- Key types:
--   - 'user': Personal keys tied to user identity (user_id NOT NULL)
--   - 'service_account': Org automation keys (user_id NULL, scoped to org_id)

CREATE TABLE api_keys (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    tenant_id BIGINT NOT NULL REFERENCES tenants(id),
    org_id UUID NOT NULL REFERENCES organizations(org_id),
    user_id UUID,  -- Nullable for service accounts; FK added in migration 000110
    name VARCHAR(255) NOT NULL,
    key_hash VARCHAR(255) NOT NULL UNIQUE,
    key_prefix VARCHAR(8) NOT NULL,
    key_type VARCHAR(20) NOT NULL DEFAULT 'user' CHECK (key_type IN ('user', 'service_account')),
    expires_at TIMESTAMPTZ,
    last_used_at TIMESTAMPTZ,
    is_active BOOLEAN NOT NULL DEFAULT true,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Performance indexes
CREATE INDEX idx_api_keys_tenant_id ON api_keys(tenant_id);
CREATE INDEX idx_api_keys_org_id ON api_keys(org_id);
CREATE INDEX idx_api_keys_user_id ON api_keys(user_id) WHERE user_id IS NOT NULL;
CREATE INDEX idx_api_keys_key_hash ON api_keys(key_hash);
CREATE INDEX idx_api_keys_key_type ON api_keys(key_type);

-- Partial unique indexes for name uniqueness per scope
-- User keys: unique name per user
CREATE UNIQUE INDEX api_keys_user_name_uq ON api_keys(user_id, name) WHERE user_id IS NOT NULL;
-- Service account keys: unique name per organization
CREATE UNIQUE INDEX api_keys_org_name_uq ON api_keys(org_id, name) WHERE user_id IS NULL;

COMMENT ON TABLE api_keys IS 'API keys for programmatic KiloCenter access';
COMMENT ON COLUMN api_keys.key_type IS 'Key scope: user (personal) or service_account (org automation)';
COMMENT ON COLUMN api_keys.key_hash IS 'SHA-256 hash of the actual key for secure storage';
COMMENT ON COLUMN api_keys.key_prefix IS 'First 8 characters of key for display/identification';
