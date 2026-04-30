-- Migration: 018_mioty_message_storage (DOWN)
-- Description: Remove MIOTY protocol message storage tables and related objects
-- Author: KiloCenter Development Team
-- Date: 2025-06-13

-- Drop triggers first
DROP TRIGGER IF EXISTS trigger_mioty_messages_updated_at ON mioty_messages;
DROP TRIGGER IF EXISTS trigger_mioty_dedup_updated_at ON mioty_message_deduplication;

-- Drop trigger function
DROP FUNCTION IF EXISTS update_mioty_message_timestamp();

-- Drop partition creation function
DROP FUNCTION IF EXISTS create_mioty_message_partition(DATE);

-- Drop all partitions (they will be dropped automatically with the parent table)
-- But we'll explicitly drop them for clarity
DO $$
DECLARE
    partition_record RECORD;
BEGIN
    FOR partition_record IN 
        SELECT schemaname, tablename 
        FROM pg_tables 
        WHERE tablename LIKE 'mioty_messages_%'
    LOOP
        EXECUTE format('DROP TABLE IF EXISTS %I.%I CASCADE', 
                       partition_record.schemaname, partition_record.tablename);
    END LOOP;
END $$;

-- Drop indexes (will be dropped with tables, but explicit for clarity)
DROP INDEX IF EXISTS idx_mioty_messages_ep_eui;
DROP INDEX IF EXISTS idx_mioty_messages_sh_addr;
DROP INDEX IF EXISTS idx_mioty_messages_tenant_time;
DROP INDEX IF EXISTS idx_mioty_messages_type;
DROP INDEX IF EXISTS idx_mioty_messages_basestation;
DROP INDEX IF EXISTS idx_mioty_messages_dedup;
DROP INDEX IF EXISTS idx_mioty_dedup_lookup;
DROP INDEX IF EXISTS idx_mioty_messages_operation;
DROP INDEX IF EXISTS idx_mioty_messages_format;
DROP INDEX IF EXISTS idx_mioty_messages_type_eui;
DROP INDEX IF EXISTS idx_mioty_subpackets_message;
DROP INDEX IF EXISTS idx_mioty_subpackets_tenant;
DROP INDEX IF EXISTS idx_mioty_bs_status_tenant;
DROP INDEX IF EXISTS idx_mioty_bs_status_basestation;
DROP INDEX IF EXISTS idx_mioty_bs_status_eui;

-- Drop tables in dependency order
DROP TABLE IF EXISTS mioty_subpackets CASCADE;
DROP TABLE IF EXISTS mioty_basestation_status CASCADE;
DROP TABLE IF EXISTS mioty_message_deduplication CASCADE;
DROP TABLE IF EXISTS mioty_messages CASCADE;

-- Drop custom types
DROP TYPE IF EXISTS interface_type;
DROP TYPE IF EXISTS message_direction;
DROP TYPE IF EXISTS message_type;
