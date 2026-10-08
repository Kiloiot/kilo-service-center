-- Restore the endpoint_keys subsystem removed by the up migration in the exact
-- shape it had at 000144. Key rows are NOT restored: the up migration refuses
-- to run unless both tables are empty (the rekey command reconciles live rows
-- into the endpoints envelopes and exports the archive first), so there is no
-- row data to bring back - recovering exported rows is done from the rekey
-- command's encrypted export file, not from this migration.
--
-- The two tables differ because of their history. endpoint_keys is the
-- BIGINT-keyed table migration 015 rebuilt; its rename-and-recreate collided
-- with the old names, so its constraints carry a "1" suffix. In particular
-- endpoint_keys_tenant_id_fkey1 is the name migration 000053's down script
-- drops before dropping the tenants table. endpoint_keys_archive was created
-- by migration 014 LIKE the pre-015 table: a UUID key, the encryption and
-- audit columns of migration 012, the original key_type vocabulary, and the
-- constraint names of that table. set_archived_at() was never dropped by the
-- up migration, so the archive trigger references it directly.

BEGIN;

CREATE SEQUENCE endpoint_keys_id_seq;

CREATE TABLE endpoint_keys (
    id BIGINT DEFAULT nextval('endpoint_keys_id_seq') CONSTRAINT endpoint_keys_id_not_null1 NOT NULL,
    endpoint_id BIGINT CONSTRAINT endpoint_keys_endpoint_id_not_null1 NOT NULL,
    tenant_id BIGINT CONSTRAINT endpoint_keys_tenant_id_not_null1 NOT NULL,
    key_type VARCHAR(20) CONSTRAINT endpoint_keys_key_type_not_null1 NOT NULL,
    key_version INTEGER DEFAULT 1 CONSTRAINT endpoint_keys_key_version_not_null1 NOT NULL,
    key_value BYTEA CONSTRAINT endpoint_keys_key_value_not_null1 NOT NULL,
    key_id VARCHAR(64),
    algorithm VARCHAR(20) DEFAULT 'AES-128',
    valid_from TIMESTAMPTZ DEFAULT NOW() CONSTRAINT endpoint_keys_valid_from_not_null1 NOT NULL,
    valid_until TIMESTAMPTZ,
    is_active BOOLEAN DEFAULT true,
    usage_count BIGINT DEFAULT 0,
    last_used_at TIMESTAMPTZ,
    rotated_from BIGINT,
    rotation_reason VARCHAR(100),
    metadata JSONB DEFAULT '{}',
    created_at TIMESTAMPTZ DEFAULT NOW() CONSTRAINT endpoint_keys_created_at_not_null1 NOT NULL,
    created_by VARCHAR(255),
    CONSTRAINT endpoint_keys_pkey1 PRIMARY KEY (id),
    CONSTRAINT endpoint_keys_algorithm_check1 CHECK (algorithm IN ('AES-128', 'AES-256')),
    CONSTRAINT endpoint_keys_key_type_check1 CHECK (key_type IN ('network', 'application', 'join', 'session')),
    CONSTRAINT endpoint_keys_endpoint_id_fkey1 FOREIGN KEY (endpoint_id) REFERENCES endpoints(id) ON DELETE CASCADE,
    CONSTRAINT endpoint_keys_tenant_id_fkey1 FOREIGN KEY (tenant_id) REFERENCES tenants(id) ON DELETE CASCADE,
    CONSTRAINT endpoint_keys_rotated_from_fkey1 FOREIGN KEY (rotated_from) REFERENCES endpoint_keys(id)
);

ALTER SEQUENCE endpoint_keys_id_seq OWNED BY endpoint_keys.id;

CREATE UNIQUE INDEX idx_endpoint_keys_unique_active
    ON endpoint_keys(endpoint_id, key_type)
    WHERE is_active = true;

CREATE TABLE endpoint_keys_archive (
    id UUID DEFAULT gen_random_uuid() CONSTRAINT endpoint_keys_id_not_null NOT NULL,
    endpoint_id BIGINT CONSTRAINT endpoint_keys_endpoint_id_not_null NOT NULL,
    tenant_id BIGINT CONSTRAINT endpoint_keys_tenant_id_not_null NOT NULL,
    key_type VARCHAR(20) CONSTRAINT endpoint_keys_key_type_not_null NOT NULL,
    key_version INTEGER DEFAULT 1 CONSTRAINT endpoint_keys_key_version_not_null NOT NULL,
    key_value BYTEA CONSTRAINT endpoint_keys_key_value_not_null NOT NULL,
    key_id VARCHAR(64),
    algorithm VARCHAR(20) DEFAULT 'AES-128',
    valid_from TIMESTAMPTZ DEFAULT NOW() CONSTRAINT endpoint_keys_valid_from_not_null NOT NULL,
    valid_until TIMESTAMPTZ,
    is_active BOOLEAN DEFAULT true,
    usage_count BIGINT DEFAULT 0,
    last_used_at TIMESTAMPTZ,
    rotated_from UUID,
    rotation_reason VARCHAR(100),
    metadata JSONB DEFAULT '{}',
    created_at TIMESTAMPTZ DEFAULT NOW() CONSTRAINT endpoint_keys_created_at_not_null NOT NULL,
    created_by VARCHAR(255),
    encryption_key_id UUID,
    encrypted_at TIMESTAMPTZ,
    encryption_metadata JSONB,
    last_modified_by VARCHAR(255),
    last_modified_at TIMESTAMPTZ,
    archived_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT endpoint_keys_archive_pkey PRIMARY KEY (id),
    CONSTRAINT endpoint_keys_algorithm_check CHECK (algorithm IN ('AES-128', 'AES-256')),
    CONSTRAINT endpoint_keys_key_type_check CHECK (key_type IN ('nwk', 'app', 'join'))
);

CREATE UNIQUE INDEX endpoint_keys_archive_endpoint_id_key_type_idx
    ON endpoint_keys_archive(endpoint_id, key_type)
    WHERE is_active = true;
CREATE INDEX endpoint_keys_archive_endpoint_id_key_type_is_active_idx
    ON endpoint_keys_archive(endpoint_id, key_type, is_active);
CREATE INDEX endpoint_keys_archive_rotated_from_idx
    ON endpoint_keys_archive(rotated_from)
    WHERE rotated_from IS NOT NULL;
CREATE INDEX endpoint_keys_archive_tenant_id_created_at_idx
    ON endpoint_keys_archive(tenant_id, created_at DESC);
CREATE INDEX endpoint_keys_archive_valid_from_valid_until_idx
    ON endpoint_keys_archive(valid_from, valid_until)
    WHERE is_active = true;
CREATE INDEX idx_endpoint_keys_archive_archived_at ON endpoint_keys_archive(archived_at);
CREATE INDEX idx_endpoint_keys_archive_endpoint_id ON endpoint_keys_archive(endpoint_id);
CREATE INDEX idx_endpoint_keys_archive_key_type ON endpoint_keys_archive(key_type);
CREATE INDEX idx_endpoint_keys_archive_tenant_id ON endpoint_keys_archive(tenant_id);

COMMENT ON TABLE endpoint_keys_archive IS 'Archive table for inactive/expired endpoint keys';
COMMENT ON COLUMN endpoint_keys_archive.key_type IS 'Type of key: nwk (network), app (application), or join';
COMMENT ON COLUMN endpoint_keys_archive.key_value IS 'Encrypted key material';
COMMENT ON COLUMN endpoint_keys_archive.key_id IS 'External identifier for key management system';
COMMENT ON COLUMN endpoint_keys_archive.usage_count IS 'Number of times this key has been used';
COMMENT ON COLUMN endpoint_keys_archive.rotated_from IS 'Previous key this was rotated from';
COMMENT ON COLUMN endpoint_keys_archive.encryption_key_id IS 'Reference to encryption key used for key_value';
COMMENT ON COLUMN endpoint_keys_archive.encrypted_at IS 'Timestamp when the key was encrypted';
COMMENT ON COLUMN endpoint_keys_archive.encryption_metadata IS 'Additional encryption metadata (IV, salt, etc.)';
COMMENT ON COLUMN endpoint_keys_archive.archived_at IS 'Timestamp when the record was archived';

CREATE TRIGGER set_endpoint_keys_archive_timestamp
    BEFORE INSERT ON endpoint_keys_archive
    FOR EACH ROW
    EXECUTE FUNCTION set_archived_at();

-- The two functions return without their triggers, matching 000144, where
-- migration 015 had already removed the endpoint_keys triggers. Their bodies
-- are the originals of migrations 006 and 013.
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

COMMIT;
