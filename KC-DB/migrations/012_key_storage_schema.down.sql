-- Rollback key storage schema

-- Remove trigger
DROP TRIGGER IF EXISTS encryption_keys_audit ON encryption_keys;

-- Remove functions
DROP FUNCTION IF EXISTS log_key_operation();
DROP FUNCTION IF EXISTS check_key_expiration();
DROP FUNCTION IF EXISTS rotate_encryption_key(UUID, UUID, VARCHAR, UUID[]);

-- Remove views
DROP VIEW IF EXISTS v_active_encryption_keys;

-- Remove encryption support from endpoint_keys (table was renamed from device_keys)
DO $$
BEGIN
    IF EXISTS (SELECT 1 FROM information_schema.tables WHERE table_schema = 'public' AND table_name = 'endpoint_keys') THEN
        ALTER TABLE endpoint_keys
            DROP COLUMN IF EXISTS encryption_key_id,
            DROP COLUMN IF EXISTS encrypted_at,
            DROP COLUMN IF EXISTS encryption_metadata;
    END IF;
END $$;

-- Drop tables in reverse dependency order
DROP TABLE IF EXISTS encrypted_fields;
DROP TABLE IF EXISTS key_usage_log;
DROP TABLE IF EXISTS encryption_keys;
