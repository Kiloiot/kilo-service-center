-- Create endpoint_keys table
-- Manages network and application keys for endpoints

CREATE TABLE endpoint_keys (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    endpoint_id BIGINT NOT NULL REFERENCES endpoints(id) ON DELETE CASCADE,
    tenant_id BIGINT NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    
    -- Key type and version
    key_type VARCHAR(20) NOT NULL CHECK (key_type IN ('nwk', 'app', 'join')),
    key_version INTEGER NOT NULL DEFAULT 1,
    
    -- Key data (encrypted at rest in production)
    key_value BYTEA NOT NULL,
    
    -- Key metadata
    key_id VARCHAR(64), -- External key identifier
    algorithm VARCHAR(20) DEFAULT 'AES-128' CHECK (algorithm IN ('AES-128', 'AES-256')),
    
    -- Validity period
    valid_from TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    valid_until TIMESTAMPTZ,
    is_active BOOLEAN DEFAULT true,
    
    -- Usage tracking
    usage_count BIGINT DEFAULT 0,
    last_used_at TIMESTAMPTZ,
    
    -- Key rotation
    rotated_from UUID REFERENCES endpoint_keys(id),
    rotation_reason VARCHAR(100),
    
    -- Metadata
    metadata JSONB DEFAULT '{}',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    created_by VARCHAR(255)
);

-- Indexes
CREATE INDEX idx_endpoint_keys_endpoint ON endpoint_keys(endpoint_id, key_type, is_active);
CREATE INDEX idx_endpoint_keys_tenant ON endpoint_keys(tenant_id, created_at DESC);
CREATE INDEX idx_endpoint_keys_validity ON endpoint_keys(valid_from, valid_until) WHERE is_active = true;
CREATE INDEX idx_endpoint_keys_rotation ON endpoint_keys(rotated_from) WHERE rotated_from IS NOT NULL;

-- Create partial unique constraint as a separate index
CREATE UNIQUE INDEX idx_endpoint_keys_unique_active 
ON endpoint_keys(endpoint_id, key_type) 
WHERE is_active = true;

-- Function to ensure only one active key per type
CREATE OR REPLACE FUNCTION ensure_single_active_key()
RETURNS TRIGGER AS $$
BEGIN
    IF NEW.is_active = true THEN
        UPDATE endpoint_keys
        SET is_active = false, valid_until = NOW()
        WHERE endpoint_id = NEW.endpoint_id
        AND key_type = NEW.key_type
        AND id != NEW.id
        AND is_active = true;
    END IF;
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

-- Trigger to maintain single active key
CREATE TRIGGER endpoint_keys_single_active
    BEFORE INSERT OR UPDATE ON endpoint_keys
    FOR EACH ROW
    WHEN (NEW.is_active = true)
    EXECUTE FUNCTION ensure_single_active_key();

-- Comments
COMMENT ON TABLE endpoint_keys IS 'Cryptographic keys for End Points';
COMMENT ON COLUMN endpoint_keys.key_type IS 'Type of key: nwk (network), app (application), or join';
COMMENT ON COLUMN endpoint_keys.key_value IS 'Encrypted key material';
COMMENT ON COLUMN endpoint_keys.key_id IS 'External identifier for key management system';
COMMENT ON COLUMN endpoint_keys.rotated_from IS 'Previous key this was rotated from';
COMMENT ON COLUMN endpoint_keys.usage_count IS 'Number of times this key has been used';
