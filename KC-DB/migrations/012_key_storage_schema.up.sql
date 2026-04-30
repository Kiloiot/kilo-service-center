-- Key storage schema for encryption support
-- Designed to support future encryption at rest implementation

-- Encryption keys metadata table
CREATE TABLE encryption_keys (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    key_id VARCHAR(255) UNIQUE NOT NULL, -- External KMS reference
    key_type VARCHAR(50) NOT NULL CHECK (key_type IN ('master', 'data', 'field', 'backup')),
    algorithm VARCHAR(50) NOT NULL DEFAULT 'AES-256-GCM',
    key_version INTEGER NOT NULL DEFAULT 1,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    created_by VARCHAR(255),
    rotated_from UUID REFERENCES encryption_keys(id),
    expires_at TIMESTAMPTZ NOT NULL,
    status VARCHAR(20) NOT NULL DEFAULT 'active' CHECK (status IN ('active', 'rotating', 'expired', 'revoked')),
    metadata JSONB
);

-- Create partial unique constraint as a separate index
CREATE UNIQUE INDEX idx_encryption_keys_unique_active 
ON encryption_keys(key_type) 
WHERE status = 'active';

CREATE INDEX idx_encryption_keys_status ON encryption_keys(status) WHERE status = 'active';
CREATE INDEX idx_encryption_keys_expires ON encryption_keys(expires_at) WHERE status = 'active';
CREATE INDEX idx_encryption_keys_type ON encryption_keys(key_type, key_version);

-- Key usage tracking
CREATE TABLE key_usage_log (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    key_id UUID NOT NULL REFERENCES encryption_keys(id),
    operation VARCHAR(50) NOT NULL CHECK (operation IN ('encrypt', 'decrypt', 'sign', 'verify', 'rotate')),
    entity_type VARCHAR(50) NOT NULL,
    entity_id UUID NOT NULL,
    performed_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    performed_by VARCHAR(255),
    success BOOLEAN NOT NULL DEFAULT true,
    error_message TEXT,
    metadata JSONB
);

CREATE INDEX idx_key_usage_log_key ON key_usage_log(key_id, performed_at DESC);
CREATE INDEX idx_key_usage_log_entity ON key_usage_log(entity_type, entity_id);
CREATE INDEX idx_key_usage_log_time ON key_usage_log(performed_at DESC);

-- Encrypted fields registry
CREATE TABLE encrypted_fields (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    table_name VARCHAR(100) NOT NULL,
    column_name VARCHAR(100) NOT NULL,
    encryption_key_id UUID NOT NULL REFERENCES encryption_keys(id),
    encryption_type VARCHAR(50) NOT NULL DEFAULT 'field' CHECK (encryption_type IN ('field', 'column', 'table')),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    status VARCHAR(20) NOT NULL DEFAULT 'active' CHECK (status IN ('active', 'migrating', 'disabled'))
);

-- Create partial unique constraint as a separate index
CREATE UNIQUE INDEX idx_encrypted_fields_unique_active 
ON encrypted_fields(table_name, column_name) 
WHERE status = 'active';

CREATE INDEX idx_encrypted_fields_table ON encrypted_fields(table_name, column_name) WHERE status = 'active';

-- Update endpoint_keys table to support encryption metadata
ALTER TABLE endpoint_keys
    ADD COLUMN encryption_key_id UUID REFERENCES encryption_keys(id),
    ADD COLUMN encrypted_at TIMESTAMPTZ,
    ADD COLUMN encryption_metadata JSONB;

-- Add encryption support to sensitive columns
COMMENT ON COLUMN endpoint_keys.encryption_key_id IS 'Reference to encryption key used for key_value';
COMMENT ON COLUMN endpoint_keys.encrypted_at IS 'Timestamp when the key was encrypted';
COMMENT ON COLUMN endpoint_keys.encryption_metadata IS 'Additional encryption metadata (IV, salt, etc.)';

-- Create function to track key rotation
CREATE OR REPLACE FUNCTION rotate_encryption_key(
    old_key_id UUID,
    new_key_id UUID,
    entity_type VARCHAR,
    entity_ids UUID[]
)
RETURNS void AS $$
DECLARE
    entity_id UUID;
BEGIN
    -- Mark old key as rotating
    UPDATE encryption_keys 
    SET status = 'rotating' 
    WHERE id = old_key_id;
    
    -- Log rotation start
    INSERT INTO key_usage_log (
        key_id, operation, entity_type, entity_id, 
        performed_by, metadata
    )
    SELECT 
        old_key_id, 'rotate', entity_type, unnest(entity_ids),
        current_user, jsonb_build_object('new_key_id', new_key_id)
    FROM unnest(entity_ids);
    
    -- Update encryption metadata would happen here in application
    -- This is just tracking the operation
    
    RAISE NOTICE 'Key rotation initiated from % to % for % entities', 
        old_key_id, new_key_id, array_length(entity_ids, 1);
END;
$$ LANGUAGE plpgsql;

-- Create view for active encryption keys
CREATE VIEW v_active_encryption_keys AS
SELECT 
    ek.id,
    ek.key_id,
    ek.key_type,
    ek.algorithm,
    ek.key_version,
    ek.created_at,
    ek.expires_at,
    ek.expires_at - CURRENT_TIMESTAMP AS time_until_expiry,
    COUNT(DISTINCT ef.id) AS encrypted_fields_count,
    COUNT(DISTINCT kul.id) AS usage_count_last_24h
FROM encryption_keys ek
LEFT JOIN encrypted_fields ef ON ef.encryption_key_id = ek.id AND ef.status = 'active'
LEFT JOIN key_usage_log kul ON kul.key_id = ek.id 
    AND kul.performed_at > CURRENT_TIMESTAMP - INTERVAL '24 hours'
WHERE ek.status = 'active'
GROUP BY ek.id;

-- Create function to check key expiration
CREATE OR REPLACE FUNCTION check_key_expiration()
RETURNS TABLE(
    key_id UUID,
    key_type VARCHAR,
    expires_in_days INTEGER,
    status VARCHAR
) AS $$
BEGIN
    RETURN QUERY
    SELECT 
        ek.id,
        ek.key_type,
        EXTRACT(DAY FROM ek.expires_at - CURRENT_TIMESTAMP)::INTEGER AS expires_in_days,
        CASE 
            WHEN ek.expires_at < CURRENT_TIMESTAMP THEN 'expired'
            WHEN ek.expires_at < CURRENT_TIMESTAMP + INTERVAL '7 days' THEN 'critical'
            WHEN ek.expires_at < CURRENT_TIMESTAMP + INTERVAL '30 days' THEN 'warning'
            ELSE 'ok'
        END AS status
    FROM encryption_keys ek
    WHERE ek.status = 'active'
    ORDER BY ek.expires_at;
END;
$$ LANGUAGE plpgsql;

-- Add security event logging for key operations
CREATE OR REPLACE FUNCTION log_key_operation()
RETURNS trigger AS $$
BEGIN
    IF TG_OP = 'INSERT' THEN
        INSERT INTO system_events (
            event_type, event_category, severity,
            title, description, metadata
        ) VALUES (
            'security.key.created', 'security', 'info',
            'Encryption key created',
            format('New %s encryption key created', NEW.key_type),
            jsonb_build_object(
                'key_id', NEW.id,
                'key_type', NEW.key_type,
                'algorithm', NEW.algorithm
            )
        );
    ELSIF TG_OP = 'UPDATE' AND OLD.status != NEW.status THEN
        INSERT INTO system_events (
            event_type, event_category, severity,
            title, description, metadata
        ) VALUES (
            'security.key.status_changed', 'security', 
            CASE NEW.status 
                WHEN 'revoked' THEN 'warning'
                WHEN 'expired' THEN 'warning'
                ELSE 'info'
            END,
            format('Encryption key status changed to %s', NEW.status),
            format('Key %s changed from %s to %s', NEW.key_id, OLD.status, NEW.status),
            jsonb_build_object(
                'key_id', NEW.id,
                'old_status', OLD.status,
                'new_status', NEW.status
            )
        );
    END IF;
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER encryption_keys_audit
    AFTER INSERT OR UPDATE ON encryption_keys
    FOR EACH ROW
    EXECUTE FUNCTION log_key_operation();

-- Add comments
COMMENT ON TABLE encryption_keys IS 'Metadata for encryption keys used in the system';
COMMENT ON TABLE key_usage_log IS 'Audit log of encryption key usage';
COMMENT ON TABLE encrypted_fields IS 'Registry of which fields are encrypted with which keys';
COMMENT ON FUNCTION rotate_encryption_key IS 'Initiate key rotation for specified entities';
COMMENT ON FUNCTION check_key_expiration IS 'Check status of active encryption keys';
COMMENT ON VIEW v_active_encryption_keys IS 'Summary view of active encryption keys and their usage';
