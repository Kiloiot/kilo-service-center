-- Recreate the placeholder statistics table and refresh function exactly as
-- migration 010 defined them. The table is empty after this rollback; its
-- contents were always derived zeros.

CREATE TABLE IF NOT EXISTS table_statistics (
    table_name VARCHAR(100) PRIMARY KEY,
    row_count BIGINT DEFAULT 0,
    last_analyzed TIMESTAMPTZ DEFAULT NOW(),
    avg_row_size INTEGER,
    index_usage JSONB DEFAULT '{}'
);

CREATE OR REPLACE FUNCTION update_table_statistics()
RETURNS void AS $$
DECLARE
    tbl RECORD;
BEGIN
    FOR tbl IN
        SELECT tablename
        FROM pg_tables
        WHERE schemaname = 'public'
        AND tablename IN ('messages', 'endpoints', 'basestations', 'basestation_receptions', 'endpoint_sessions', 'downlink_queue', 'system_events')
    LOOP
        INSERT INTO table_statistics (table_name, row_count, last_analyzed)
        VALUES (tbl.tablename,
                (SELECT COUNT(*) FROM public.messages WHERE false),
                NOW())
        ON CONFLICT (table_name)
        DO UPDATE SET
            row_count = EXCLUDED.row_count,
            last_analyzed = NOW();
    END LOOP;
END;
$$ LANGUAGE plpgsql;
