-- Set up additional partitioning utility functions for messages table
-- The messages table is already partitioned in the initial schema

-- Create function to automatically create partitions (trigger function)
CREATE OR REPLACE FUNCTION auto_create_monthly_partition()
RETURNS trigger AS $$
DECLARE
    partition_date date;
    partition_name text;
    start_date date;
    end_date date;
BEGIN
    partition_date := date_trunc('month', NEW.received_at)::date;
    partition_name := 'messages_' || to_char(partition_date, 'YYYY_MM');
    start_date := partition_date;
    end_date := start_date + interval '1 month';

    -- Check if partition already exists
    IF NOT EXISTS (
        SELECT 1 FROM pg_tables 
        WHERE tablename = partition_name
    ) THEN
        EXECUTE format('CREATE TABLE %I PARTITION OF messages 
                        FOR VALUES FROM (%L) TO (%L)', 
                        partition_name, start_date, end_date);
    END IF;
    
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

-- Create maintenance function to create future partitions
CREATE OR REPLACE FUNCTION maintain_message_partitions(months_ahead integer DEFAULT 3)
RETURNS void AS $$
DECLARE
    curr_date date;
    future_date date;
    partition_name text;
    start_date date;
    end_date date;
BEGIN
    curr_date := date_trunc('month', CURRENT_DATE)::date;
    future_date := (curr_date + (months_ahead || ' months')::interval)::date;

    WHILE curr_date <= future_date LOOP
        partition_name := 'messages_' || to_char(curr_date, 'YYYY_MM');
        start_date := curr_date;
        end_date := start_date + interval '1 month';
        
        -- Check if partition already exists
        IF NOT EXISTS (
            SELECT 1 FROM pg_tables 
            WHERE tablename = partition_name
        ) THEN
            EXECUTE format('CREATE TABLE %I PARTITION OF messages 
                            FOR VALUES FROM (%L) TO (%L)', 
                            partition_name, start_date, end_date);
        END IF;
        
        curr_date := (curr_date + interval '1 month')::date;
    END LOOP;
END;
$$ LANGUAGE plpgsql;

-- Create function to drop old partitions (for archival)
CREATE OR REPLACE FUNCTION drop_old_message_partitions(months_to_keep integer DEFAULT 3)
RETURNS void AS $$
DECLARE
    partition_record record;
    cutoff_date date;
BEGIN
    cutoff_date := (date_trunc('month', CURRENT_DATE) - (months_to_keep || ' months')::interval)::date;

    FOR partition_record IN
        SELECT
            schemaname,
            tablename
        FROM pg_tables
        WHERE schemaname = 'public'
        AND tablename LIKE 'messages_%'
        AND tablename ~ '_\d{4}_\d{2}$'
    LOOP
        -- Extract date from partition name
        IF to_date(right(partition_record.tablename, 7), 'YYYY_MM') < cutoff_date THEN
            -- Archive data before dropping (implement based on archival strategy)
            RAISE NOTICE 'Dropping old partition: %', partition_record.tablename;
            EXECUTE format('DROP TABLE %I.%I', partition_record.schemaname, partition_record.tablename);
        END IF;
    END LOOP;
END;
$$ LANGUAGE plpgsql;

-- Add comments
COMMENT ON FUNCTION auto_create_monthly_partition IS 'Trigger function to automatically create monthly partitions';
COMMENT ON FUNCTION maintain_message_partitions IS 'Call this monthly to ensure future partitions exist';
COMMENT ON FUNCTION drop_old_message_partitions IS 'Call this monthly to archive and drop old partitions';
