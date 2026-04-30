-- Create tenants table for tenant management (if not exists from earlier migration)
CREATE TABLE IF NOT EXISTS tenants (
    id BIGSERIAL PRIMARY KEY,
    name VARCHAR(255) NOT NULL UNIQUE,
    description TEXT,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);

-- Add status column if it doesn't exist (table may exist from migration 001 with is_active instead)
ALTER TABLE tenants
ADD COLUMN IF NOT EXISTS status VARCHAR(50) DEFAULT 'active';

-- Update status to NOT NULL after adding it
DO $$
BEGIN
    -- Only set NOT NULL if the column exists and has no nulls
    IF EXISTS (SELECT 1 FROM information_schema.columns
               WHERE table_name = 'tenants' AND column_name = 'status') THEN
        ALTER TABLE tenants ALTER COLUMN status SET DEFAULT 'active';
        UPDATE tenants SET status = 'active' WHERE status IS NULL;
        ALTER TABLE tenants ALTER COLUMN status SET NOT NULL;
    END IF;
END $$;

-- Create indexes
CREATE INDEX IF NOT EXISTS idx_tenants_status ON tenants(status);
CREATE INDEX IF NOT EXISTS idx_tenants_name ON tenants(name);

-- Add unique constraint on name (idempotent)
DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM pg_constraint
        WHERE conname = 'tenants_name_key'
        AND conrelid = 'tenants'::regclass
    ) THEN
        ALTER TABLE tenants ADD CONSTRAINT tenants_name_key UNIQUE (name);
    END IF;
END $$;

-- Insert default tenant if it doesn't exist
INSERT INTO tenants (id, name, description, status, created_at, updated_at)
VALUES (1, 'Default', 'Default tenant for single-tenant installations', 'active', NOW(), NOW())
ON CONFLICT (id) DO NOTHING;

-- Restore FK constraints that existed before migration 053 (idempotent)
-- These constraints were created in migrations 001-051 and need to be restored
-- if the tenants table was dropped and recreated

-- 1. endpoints_tenant_id_fkey (migration 001:28)
DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM pg_constraint
        WHERE conname = 'endpoints_tenant_id_fkey'
        AND conrelid = 'endpoints'::regclass
    ) THEN
        ALTER TABLE endpoints
            ADD CONSTRAINT endpoints_tenant_id_fkey
            FOREIGN KEY (tenant_id) REFERENCES tenants(id) ON DELETE CASCADE;
    END IF;
END $$;

-- 2. basestations_tenant_id_fkey (migration 001:55)
DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM pg_constraint
        WHERE conname = 'basestations_tenant_id_fkey'
        AND conrelid = 'basestations'::regclass
    ) THEN
        ALTER TABLE basestations
            ADD CONSTRAINT basestations_tenant_id_fkey
            FOREIGN KEY (tenant_id) REFERENCES tenants(id) ON DELETE CASCADE;
    END IF;
END $$;

-- 3. downlink_queue_tenant_id_fkey (migration 001:128)
DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM pg_constraint
        WHERE conname = 'downlink_queue_tenant_id_fkey'
        AND conrelid = 'downlink_queue'::regclass
    ) THEN
        ALTER TABLE downlink_queue
            ADD CONSTRAINT downlink_queue_tenant_id_fkey
            FOREIGN KEY (tenant_id) REFERENCES tenants(id) ON DELETE CASCADE;
    END IF;
END $$;

-- 4. roaming_agreements_tenant_id_fkey (migration 001:164)
DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM pg_constraint
        WHERE conname = 'roaming_agreements_tenant_id_fkey'
        AND conrelid = 'roaming_agreements'::regclass
    ) THEN
        ALTER TABLE roaming_agreements
            ADD CONSTRAINT roaming_agreements_tenant_id_fkey
            FOREIGN KEY (tenant_id) REFERENCES tenants(id) ON DELETE CASCADE;
    END IF;
END $$;

-- 5. basestation_receptions_tenant_id_fkey (migration 004:8)
DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM pg_constraint
        WHERE conname = 'basestation_receptions_tenant_id_fkey'
        AND conrelid = 'basestation_receptions'::regclass
    ) THEN
        ALTER TABLE basestation_receptions
            ADD CONSTRAINT basestation_receptions_tenant_id_fkey
            FOREIGN KEY (tenant_id) REFERENCES tenants(id) ON DELETE CASCADE;
    END IF;
END $$;

-- 6. endpoint_sessions_tenant_id_fkey1 (migration 005:7)
DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM pg_constraint
        WHERE conname = 'endpoint_sessions_tenant_id_fkey1'
        AND conrelid = 'endpoint_sessions'::regclass
    ) THEN
        ALTER TABLE endpoint_sessions
            ADD CONSTRAINT endpoint_sessions_tenant_id_fkey1
            FOREIGN KEY (tenant_id) REFERENCES tenants(id) ON DELETE CASCADE;
    END IF;
END $$;

-- 7. endpoint_keys_tenant_id_fkey1 (migration 006:7)
DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM pg_constraint
        WHERE conname = 'endpoint_keys_tenant_id_fkey1'
        AND conrelid = 'endpoint_keys'::regclass
    ) THEN
        ALTER TABLE endpoint_keys
            ADD CONSTRAINT endpoint_keys_tenant_id_fkey1
            FOREIGN KEY (tenant_id) REFERENCES tenants(id) ON DELETE CASCADE;
    END IF;
END $$;

-- 8. basestation_sessions_tenant_id_fkey1 (migration 007:7)
DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM pg_constraint
        WHERE conname = 'basestation_sessions_tenant_id_fkey1'
        AND conrelid = 'basestation_sessions'::regclass
    ) THEN
        ALTER TABLE basestation_sessions
            ADD CONSTRAINT basestation_sessions_tenant_id_fkey1
            FOREIGN KEY (tenant_id) REFERENCES tenants(id) ON DELETE CASCADE;
    END IF;
END $$;

-- 9. system_events_tenant_id_fkey (migration 008:6)
DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM pg_constraint
        WHERE conname = 'system_events_tenant_id_fkey'
        AND conrelid = 'system_events'::regclass
    ) THEN
        ALTER TABLE system_events
            ADD CONSTRAINT system_events_tenant_id_fkey
            FOREIGN KEY (tenant_id) REFERENCES tenants(id) ON DELETE CASCADE;
    END IF;
END $$;

-- 10. mioty_subpackets_tenant_id_fkey (migration 018:120)
DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM pg_constraint
        WHERE conname = 'mioty_subpackets_tenant_id_fkey'
        AND conrelid = 'mioty_subpackets'::regclass
    ) THEN
        ALTER TABLE mioty_subpackets
            ADD CONSTRAINT mioty_subpackets_tenant_id_fkey
            FOREIGN KEY (tenant_id) REFERENCES tenants(id) ON DELETE CASCADE;
    END IF;
END $$;

-- 11. mioty_basestation_status_tenant_id_fkey (migration 018:144)
DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM pg_constraint
        WHERE conname = 'mioty_basestation_status_tenant_id_fkey'
        AND conrelid = 'mioty_basestation_status'::regclass
    ) THEN
        ALTER TABLE mioty_basestation_status
            ADD CONSTRAINT mioty_basestation_status_tenant_id_fkey
            FOREIGN KEY (tenant_id) REFERENCES tenants(id) ON DELETE CASCADE;
    END IF;
END $$;

-- 12. mioty_message_deduplication_tenant_id_fkey (migration 018:179)
DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM pg_constraint
        WHERE conname = 'mioty_message_deduplication_tenant_id_fkey'
        AND conrelid = 'mioty_message_deduplication'::regclass
    ) THEN
        ALTER TABLE mioty_message_deduplication
            ADD CONSTRAINT mioty_message_deduplication_tenant_id_fkey
            FOREIGN KEY (tenant_id) REFERENCES tenants(id) ON DELETE CASCADE;
    END IF;
END $$;

-- 13. dl_rx_status_tenant_id_fkey (migration 000045:6)
DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM pg_constraint
        WHERE conname = 'dl_rx_status_tenant_id_fkey'
        AND conrelid = 'dl_rx_status'::regclass
    ) THEN
        ALTER TABLE dl_rx_status
            ADD CONSTRAINT dl_rx_status_tenant_id_fkey
            FOREIGN KEY (tenant_id) REFERENCES tenants(id) ON DELETE CASCADE;
    END IF;
END $$;