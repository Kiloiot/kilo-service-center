-- Add security-related constraints and policies

-- Add audit columns to sensitive tables
ALTER TABLE endpoints 
    ADD COLUMN IF NOT EXISTS last_modified_by VARCHAR(255),
    ADD COLUMN IF NOT EXISTS last_modified_at TIMESTAMPTZ;

ALTER TABLE endpoint_keys
    ADD COLUMN IF NOT EXISTS last_modified_by VARCHAR(255),
    ADD COLUMN IF NOT EXISTS last_modified_at TIMESTAMPTZ;

ALTER TABLE basestations
    ADD COLUMN IF NOT EXISTS last_modified_by VARCHAR(255),
    ADD COLUMN IF NOT EXISTS last_modified_at TIMESTAMPTZ;

-- Create function to automatically update audit fields
CREATE OR REPLACE FUNCTION update_audit_fields()
RETURNS trigger AS $$
BEGIN
    NEW.last_modified_at = CURRENT_TIMESTAMP;
    -- last_modified_by should be set by application
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

-- Add audit triggers
CREATE TRIGGER endpoints_audit_update
    BEFORE UPDATE ON endpoints
    FOR EACH ROW
    EXECUTE FUNCTION update_audit_fields();

CREATE TRIGGER endpoint_keys_audit_update
    BEFORE UPDATE ON endpoint_keys
    FOR EACH ROW
    EXECUTE FUNCTION update_audit_fields();

CREATE TRIGGER basestations_audit_update
    BEFORE UPDATE ON basestations
    FOR EACH ROW
    EXECUTE FUNCTION update_audit_fields();

-- Add constraints for data integrity
ALTER TABLE messages
    ADD CONSTRAINT messages_rssi_range CHECK (rssi >= -150 AND rssi <= 0),
    ADD CONSTRAINT messages_snr_range CHECK (snr >= -20 AND snr <= 30),
    ADD CONSTRAINT messages_frequency_positive CHECK (frequency > 0);

ALTER TABLE downlink_queue
    ADD CONSTRAINT downlink_queue_priority_range CHECK (priority >= 0 AND priority <= 10),
    ADD CONSTRAINT downlink_queue_retry_limit CHECK (retry_count >= 0 AND retry_count <= 10),
    ADD CONSTRAINT downlink_queue_schedule_order CHECK (
        (earliest_at IS NULL AND latest_at IS NULL) OR 
        (earliest_at IS NOT NULL AND latest_at IS NOT NULL AND earliest_at <= latest_at)
    );

-- Add constraints for endpoint security
ALTER TABLE endpoints
    ADD CONSTRAINT endpoints_eui_length CHECK (length(eui) = 8);

ALTER TABLE basestations
    ADD CONSTRAINT basestations_eui_length CHECK (length(eui) = 8);

-- Create function to validate key format
CREATE OR REPLACE FUNCTION validate_key_format()
RETURNS trigger AS $$
BEGIN
    -- Validate key format based on key type
    IF NEW.key_type = 'network_key' OR NEW.key_type = 'app_key' THEN
        IF length(NEW.key_value) != 16 THEN
            RAISE EXCEPTION 'MIOTY keys must be exactly 16 bytes';
        END IF;
    END IF;
    
    -- Ensure only one active key per type per endpoint
    IF NEW.status = 'active' THEN
        IF EXISTS (
            SELECT 1 FROM endpoint_keys 
            WHERE endpoint_id = NEW.endpoint_id 
            AND key_type = NEW.key_type 
            AND status = 'active'
            AND id != COALESCE(NEW.id, '00000000-0000-0000-0000-000000000000'::uuid)
        ) THEN
            RAISE EXCEPTION 'Only one active % allowed per endpoint', NEW.key_type;
        END IF;
    END IF;
    
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER endpoint_keys_validate
    BEFORE INSERT OR UPDATE ON endpoint_keys
    FOR EACH ROW
    EXECUTE FUNCTION validate_key_format();

-- Add session security constraints
ALTER TABLE endpoint_sessions
    ADD CONSTRAINT endpoint_sessions_counters CHECK (
        packet_cnt >= 0
    );

ALTER TABLE basestation_sessions
    ADD CONSTRAINT basestation_sessions_duration CHECK (
        (ended_at IS NULL) OR (ended_at >= started_at)
    );

-- Create function to enforce tenant isolation
CREATE OR REPLACE FUNCTION enforce_tenant_isolation()
RETURNS trigger AS $$
BEGIN
    -- For INSERT operations
    IF TG_OP = 'INSERT' THEN
        -- Verify endpoint belongs to same tenant (for messages)
        IF TG_TABLE_NAME = 'messages' THEN
            IF NOT EXISTS (
                SELECT 1 FROM endpoints 
                WHERE eui = NEW.ep_eui 
                AND tenant_id = NEW.tenant_id
            ) THEN
                RAISE EXCEPTION 'Endpoint % does not belong to tenant %', 
                    NEW.ep_eui, NEW.tenant_id;
            END IF;
        END IF;
        
        -- Verify basestation belongs to same tenant (for messages)
        IF TG_TABLE_NAME = 'messages' AND NEW.bs_eui IS NOT NULL THEN
            IF NOT EXISTS (
                SELECT 1 FROM basestations 
                WHERE eui = NEW.bs_eui 
                AND tenant_id = NEW.tenant_id
            ) THEN
                RAISE EXCEPTION 'Basestation % does not belong to tenant %', 
                    NEW.bs_eui, NEW.tenant_id;
            END IF;
        END IF;
    END IF;
    
    -- For UPDATE operations, prevent changing tenant_id
    IF TG_OP = 'UPDATE' THEN
        IF OLD.tenant_id != NEW.tenant_id THEN
            RAISE EXCEPTION 'Cannot change tenant_id';
        END IF;
    END IF;
    
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

-- Apply tenant isolation to critical tables
CREATE TRIGGER messages_tenant_isolation
    BEFORE INSERT OR UPDATE ON messages
    FOR EACH ROW
    EXECUTE FUNCTION enforce_tenant_isolation();

CREATE TRIGGER endpoints_tenant_isolation
    BEFORE UPDATE ON endpoints
    FOR EACH ROW
    EXECUTE FUNCTION enforce_tenant_isolation();

CREATE TRIGGER basestations_tenant_isolation
    BEFORE UPDATE ON basestations
    FOR EACH ROW
    EXECUTE FUNCTION enforce_tenant_isolation();

-- Create security view for monitoring access patterns
CREATE VIEW v_security_access_patterns AS
WITH recent_access AS (
    SELECT 
        tenant_id,
        'message' as entity_type,
        COUNT(*) as access_count,
        COUNT(DISTINCT ep_eui) as unique_endpoints,
        COUNT(DISTINCT bs_eui) as unique_basestations,
        MIN(created_at) as first_access,
        MAX(created_at) as last_access
    FROM messages
    WHERE created_at > CURRENT_TIMESTAMP - INTERVAL '1 hour'
    GROUP BY tenant_id
)
SELECT 
    t.name as tenant_name,
    ra.*,
    CASE 
        WHEN ra.access_count > 10000 THEN 'high'
        WHEN ra.access_count > 1000 THEN 'medium'
        ELSE 'normal'
    END as activity_level
FROM recent_access ra
JOIN tenants t ON t.id = ra.tenant_id
ORDER BY ra.access_count DESC;

-- Create function to detect anomalous access patterns
CREATE OR REPLACE FUNCTION detect_anomalous_access()
RETURNS TABLE(
    tenant_id UUID,
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
            'basestation_eui', b.eui,
            'connection_count', COUNT(bs.id),
            'time_window', '1 hour'
        ) as details
    FROM basestations b
    JOIN basestation_sessions bs ON bs.basestation_id = b.id
    WHERE bs.started_at > CURRENT_TIMESTAMP - INTERVAL '1 hour'
    GROUP BY b.tenant_id, b.eui
    HAVING COUNT(bs.id) > 10;
END;
$$ LANGUAGE plpgsql;

-- Add comments
COMMENT ON FUNCTION update_audit_fields IS 'Automatically update audit timestamp on record modification';
COMMENT ON FUNCTION validate_key_format IS 'Validate cryptographic key format and enforce uniqueness';
COMMENT ON FUNCTION enforce_tenant_isolation IS 'Ensure data remains within tenant boundaries';
COMMENT ON FUNCTION detect_anomalous_access IS 'Detect potential security issues through access patterns';
COMMENT ON VIEW v_security_access_patterns IS 'Monitor data access patterns for security analysis';
