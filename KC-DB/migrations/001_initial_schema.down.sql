-- Drop all tables and extensions created in initial schema
-- This is the down migration for 001_initial_schema.up.sql

-- Drop triggers first (before functions they depend on)
DO $$
BEGIN
    IF EXISTS (SELECT 1 FROM information_schema.tables WHERE table_schema = 'public' AND table_name = 'tenants') THEN
        DROP TRIGGER IF EXISTS update_tenants_updated_at ON tenants;
    END IF;

    IF EXISTS (SELECT 1 FROM information_schema.tables WHERE table_schema = 'public' AND table_name = 'endpoints') THEN
        DROP TRIGGER IF EXISTS update_endpoints_updated_at ON endpoints;
    END IF;

    IF EXISTS (SELECT 1 FROM information_schema.tables WHERE table_schema = 'public' AND table_name = 'basestations') THEN
        DROP TRIGGER IF EXISTS update_basestations_updated_at ON basestations;
    END IF;

    IF EXISTS (SELECT 1 FROM information_schema.tables WHERE table_schema = 'public' AND table_name = 'downlink_queue') THEN
        DROP TRIGGER IF EXISTS update_downlink_queue_updated_at ON downlink_queue;
    END IF;

    IF EXISTS (SELECT 1 FROM information_schema.tables WHERE table_schema = 'public' AND table_name = 'roaming_agreements') THEN
        DROP TRIGGER IF EXISTS update_roaming_agreements_updated_at ON roaming_agreements;
    END IF;
END $$;

-- Now drop functions (after triggers are dropped)
DROP FUNCTION IF EXISTS update_updated_at_column();
DROP FUNCTION IF EXISTS archive_old_messages(integer);
DROP FUNCTION IF EXISTS create_monthly_partition();

-- Drop tables in reverse dependency order (only tables created by migration 001)
DROP TABLE IF EXISTS roaming_agreements;
DROP TABLE IF EXISTS downlink_queue;
DROP TABLE IF EXISTS messages_archive CASCADE;  -- CASCADE to drop any partitions
DROP TABLE IF EXISTS messages CASCADE;  -- CASCADE to drop any partitions
DROP TABLE IF EXISTS endpoints;
DROP TABLE IF EXISTS basestations;
DROP TABLE IF EXISTS tenants;

-- Drop extensions (only if no other objects depend on them)
-- Note: Be careful with extensions as other databases might use them
-- DROP EXTENSION IF EXISTS "hstore";
-- DROP EXTENSION IF EXISTS "uuid-ossp";
