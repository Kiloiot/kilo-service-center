-- Drop the key-storage subsystem from migrations 012 and 013.
--
-- Key material is encrypted at rest through the pkg/keycrypto envelope since
-- 000143, and the endpoint_keys table that 012 extended was dropped in
-- 000145. encryption_keys, key_usage_log and encrypted_fields never received
-- a row from any service, their helper functions have no caller, and the
-- audit trigger inserts into a system_events column that does not exist.
-- v_security_access_patterns (013) was already removed when 000047 dropped
-- the messages table with CASCADE, so it is dropped here only if a database
-- still has it. The tables must be empty; any row aborts the migration.

DO $$
DECLARE
    row_count bigint;
BEGIN
    SELECT (SELECT count(*) FROM encryption_keys)
         + (SELECT count(*) FROM key_usage_log)
         + (SELECT count(*) FROM encrypted_fields)
    INTO row_count;
    IF row_count > 0 THEN
        RAISE EXCEPTION 'key storage tables hold % row(s); the subsystem was expected to be unused', row_count;
    END IF;
END $$;

DROP TRIGGER IF EXISTS encryption_keys_audit ON encryption_keys;

DROP VIEW IF EXISTS v_active_encryption_keys;
DROP VIEW IF EXISTS v_security_access_patterns;

DROP FUNCTION IF EXISTS rotate_encryption_key(UUID, UUID, VARCHAR, UUID[]);
DROP FUNCTION IF EXISTS check_key_expiration();
DROP FUNCTION IF EXISTS log_key_operation();
DROP FUNCTION IF EXISTS detect_anomalous_access();

DROP INDEX IF EXISTS idx_encrypted_fields_table;
DROP INDEX IF EXISTS idx_encrypted_fields_unique_active;
DROP INDEX IF EXISTS idx_key_usage_log_time;
DROP INDEX IF EXISTS idx_key_usage_log_entity;
DROP INDEX IF EXISTS idx_key_usage_log_key;
DROP INDEX IF EXISTS idx_encryption_keys_type;
DROP INDEX IF EXISTS idx_encryption_keys_expires;
DROP INDEX IF EXISTS idx_encryption_keys_status;
DROP INDEX IF EXISTS idx_encryption_keys_unique_active;

DROP TABLE IF EXISTS encrypted_fields;
DROP TABLE IF EXISTS key_usage_log;
DROP TABLE IF EXISTS encryption_keys;
