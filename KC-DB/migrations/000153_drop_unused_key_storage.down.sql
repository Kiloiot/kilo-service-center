-- Recreate the key-storage tables, indexes, functions, view and trigger that
-- survived until the up migration, as migrations 012 and 013 defined them,
-- with the three defects of the originals repaired so the restored objects
-- run: the audit trigger writes system_events.data (the 012 text named a
-- metadata column that never existed), rotate_encryption_key expands the
-- entity list once instead of as a cross product, and detect_anomalous_access
-- reports BIGINT tenant ids over the bs_eui column the schema actually has,
-- and check_key_expiration returns the VARCHAR status its signature declares.
-- endpoint_keys is not touched (000145 dropped it) and
-- v_security_access_patterns is not recreated (000047 already removed it).
-- The tables come back empty; the up migration only proceeds when they are.

CREATE TABLE IF NOT EXISTS encryption_keys (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    key_id VARCHAR(255) UNIQUE NOT NULL,
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

CREATE UNIQUE INDEX IF NOT EXISTS idx_encryption_keys_unique_active
ON encryption_keys(key_type)
WHERE status = 'active';

CREATE INDEX IF NOT EXISTS idx_encryption_keys_status ON encryption_keys(status) WHERE status = 'active';
CREATE INDEX IF NOT EXISTS idx_encryption_keys_expires ON encryption_keys(expires_at) WHERE status = 'active';
CREATE INDEX IF NOT EXISTS idx_encryption_keys_type ON encryption_keys(key_type, key_version);

CREATE TABLE IF NOT EXISTS key_usage_log (
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

CREATE INDEX IF NOT EXISTS idx_key_usage_log_key ON key_usage_log(key_id, performed_at DESC);
CREATE INDEX IF NOT EXISTS idx_key_usage_log_entity ON key_usage_log(entity_type, entity_id);
CREATE INDEX IF NOT EXISTS idx_key_usage_log_time ON key_usage_log(performed_at DESC);

CREATE TABLE IF NOT EXISTS encrypted_fields (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    table_name VARCHAR(100) NOT NULL,
    column_name VARCHAR(100) NOT NULL,
    encryption_key_id UUID NOT NULL REFERENCES encryption_keys(id),
    encryption_type VARCHAR(50) NOT NULL DEFAULT 'field' CHECK (encryption_type IN ('field', 'column', 'table')),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    status VARCHAR(20) NOT NULL DEFAULT 'active' CHECK (status IN ('active', 'migrating', 'disabled'))
);

CREATE UNIQUE INDEX IF NOT EXISTS idx_encrypted_fields_unique_active
ON encrypted_fields(table_name, column_name)
WHERE status = 'active';

CREATE INDEX IF NOT EXISTS idx_encrypted_fields_table ON encrypted_fields(table_name, column_name) WHERE status = 'active';

CREATE OR REPLACE FUNCTION rotate_encryption_key(
    old_key_id UUID,
    new_key_id UUID,
    entity_type VARCHAR,
    entity_ids UUID[]
)
RETURNS void AS $$
BEGIN
    UPDATE encryption_keys
    SET status = 'rotating'
    WHERE id = old_key_id;

    INSERT INTO key_usage_log (
        key_id, operation, entity_type, entity_id,
        performed_by, metadata
    )
    SELECT
        old_key_id, 'rotate', entity_type, ids.entity_id,
        current_user, jsonb_build_object('new_key_id', new_key_id)
    FROM unnest(entity_ids) AS ids(entity_id);

    RAISE NOTICE 'Key rotation initiated from % to % for % entities',
        old_key_id, new_key_id, array_length(entity_ids, 1);
END;
$$ LANGUAGE plpgsql;

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
        END::VARCHAR AS status
    FROM encryption_keys ek
    WHERE ek.status = 'active'
    ORDER BY ek.expires_at;
END;
$$ LANGUAGE plpgsql;

CREATE OR REPLACE FUNCTION log_key_operation()
RETURNS trigger AS $$
BEGIN
    IF TG_OP = 'INSERT' THEN
        INSERT INTO system_events (
            event_type, event_category, severity,
            title, description, data
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
            title, description, data
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

CREATE OR REPLACE FUNCTION detect_anomalous_access()
RETURNS TABLE(
    tenant_id BIGINT,
    anomaly_type VARCHAR,
    severity VARCHAR,
    details JSONB
) AS $$
BEGIN
    -- Check for mass data access
    RETURN QUERY
    SELECT
        m.tenant_id,
        'mass_data_access'::VARCHAR as anomaly_type,
        'warning'::VARCHAR as severity,
        jsonb_build_object(
            'message_count', COUNT(*),
            'time_window', '5 minutes',
            'threshold', 1000
        ) as details
    FROM messages m
    WHERE m.created_at > CURRENT_TIMESTAMP - INTERVAL '5 minutes'
    GROUP BY m.tenant_id
    HAVING COUNT(*) > 1000;

    -- Check for unusual basestation activity
    RETURN QUERY
    SELECT
        b.tenant_id,
        'unusual_basestation_activity'::VARCHAR as anomaly_type,
        'info'::VARCHAR as severity,
        jsonb_build_object(
            'basestation_eui', encode(b.bs_eui, 'hex'),
            'connection_count', COUNT(bs.id),
            'time_window', '1 hour'
        ) as details
    FROM basestations b
    JOIN basestation_sessions bs ON bs.basestation_id = b.id
    WHERE bs.started_at > CURRENT_TIMESTAMP - INTERVAL '1 hour'
    GROUP BY b.tenant_id, b.bs_eui
    HAVING COUNT(bs.id) > 10;
END;
$$ LANGUAGE plpgsql;
