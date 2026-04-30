-- Rollback security constraints

-- Remove views first
DROP VIEW IF EXISTS v_security_access_patterns;

-- Remove triggers BEFORE functions (triggers depend on functions)
-- Guard with table existence checks since tables may have been dropped by earlier migrations
DO $$
BEGIN
    -- Drop triggers only if tables exist
    IF EXISTS (SELECT 1 FROM information_schema.tables WHERE table_schema = 'public' AND table_name = 'messages') THEN
        DROP TRIGGER IF EXISTS messages_tenant_isolation ON messages;
    END IF;

    IF EXISTS (SELECT 1 FROM information_schema.tables WHERE table_schema = 'public' AND table_name = 'endpoints') THEN
        DROP TRIGGER IF EXISTS endpoints_tenant_isolation ON endpoints;
        DROP TRIGGER IF EXISTS endpoints_audit_update ON endpoints;
    END IF;

    IF EXISTS (SELECT 1 FROM information_schema.tables WHERE table_schema = 'public' AND table_name = 'basestations') THEN
        DROP TRIGGER IF EXISTS basestations_tenant_isolation ON basestations;
        DROP TRIGGER IF EXISTS basestations_audit_update ON basestations;
    END IF;

    IF EXISTS (SELECT 1 FROM information_schema.tables WHERE table_schema = 'public' AND table_name = 'endpoint_keys') THEN
        DROP TRIGGER IF EXISTS endpoint_keys_validate ON endpoint_keys;
        DROP TRIGGER IF EXISTS endpoint_keys_audit_update ON endpoint_keys;
    END IF;
END $$;

-- Now remove functions (after triggers are dropped)
DROP FUNCTION IF EXISTS detect_anomalous_access();
DROP FUNCTION IF EXISTS enforce_tenant_isolation();
DROP FUNCTION IF EXISTS validate_key_format();
DROP FUNCTION IF EXISTS update_audit_fields();

-- Remove constraints (guard with table existence checks)
DO $$
BEGIN
    IF EXISTS (SELECT 1 FROM information_schema.tables WHERE table_schema = 'public' AND table_name = 'messages') THEN
        ALTER TABLE messages
            DROP CONSTRAINT IF EXISTS messages_rssi_range,
            DROP CONSTRAINT IF EXISTS messages_snr_range,
            DROP CONSTRAINT IF EXISTS messages_frequency_positive;
    END IF;

    IF EXISTS (SELECT 1 FROM information_schema.tables WHERE table_schema = 'public' AND table_name = 'downlink_queue') THEN
        ALTER TABLE downlink_queue
            DROP CONSTRAINT IF EXISTS downlink_queue_priority_range,
            DROP CONSTRAINT IF EXISTS downlink_queue_retry_limit,
            DROP CONSTRAINT IF EXISTS downlink_queue_schedule_order;
    END IF;

    IF EXISTS (SELECT 1 FROM information_schema.tables WHERE table_schema = 'public' AND table_name = 'endpoints') THEN
        ALTER TABLE endpoints
            DROP CONSTRAINT IF EXISTS endpoints_eui_length;
    END IF;

    IF EXISTS (SELECT 1 FROM information_schema.tables WHERE table_schema = 'public' AND table_name = 'basestations') THEN
        ALTER TABLE basestations
            DROP CONSTRAINT IF EXISTS basestations_eui_length;
    END IF;

    IF EXISTS (SELECT 1 FROM information_schema.tables WHERE table_schema = 'public' AND table_name = 'endpoint_sessions') THEN
        ALTER TABLE endpoint_sessions
            DROP CONSTRAINT IF EXISTS endpoint_sessions_counters;
    END IF;

    IF EXISTS (SELECT 1 FROM information_schema.tables WHERE table_schema = 'public' AND table_name = 'basestation_sessions') THEN
        ALTER TABLE basestation_sessions
            DROP CONSTRAINT IF EXISTS basestation_sessions_duration;
    END IF;
END $$;

-- Remove audit columns (guard with table existence checks)
DO $$
BEGIN
    IF EXISTS (SELECT 1 FROM information_schema.tables WHERE table_schema = 'public' AND table_name = 'endpoints') THEN
        ALTER TABLE endpoints
            DROP COLUMN IF EXISTS last_modified_by,
            DROP COLUMN IF EXISTS last_modified_at;
    END IF;

    IF EXISTS (SELECT 1 FROM information_schema.tables WHERE table_schema = 'public' AND table_name = 'endpoint_keys') THEN
        ALTER TABLE endpoint_keys
            DROP COLUMN IF EXISTS last_modified_by,
            DROP COLUMN IF EXISTS last_modified_at;
    END IF;

    IF EXISTS (SELECT 1 FROM information_schema.tables WHERE table_schema = 'public' AND table_name = 'basestations') THEN
        ALTER TABLE basestations
            DROP COLUMN IF EXISTS last_modified_by,
            DROP COLUMN IF EXISTS last_modified_at;
    END IF;
END $$;
