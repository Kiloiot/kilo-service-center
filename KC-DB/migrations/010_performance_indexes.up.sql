-- Create MIOTY-specific performance indexes and optimizations

-- Messages table: Add indexes for MIOTY-specific queries (only new ones, avoiding duplicates)
CREATE INDEX idx_messages_packet_counter ON messages(ep_eui, packet_counter DESC)
    WHERE packet_counter IS NOT NULL;
CREATE INDEX idx_messages_uplink_mode ON messages(tenant_id, uplink_mode, received_at DESC)
    WHERE uplink_mode IS NOT NULL;
CREATE INDEX idx_messages_dl_flags ON messages(ep_eui, dl_open, res_exp, dl_ack)
    WHERE dl_open = true OR res_exp = true OR dl_ack = true;

-- Endpoints table: Add indexes for attachment and status queries (only new ones)
CREATE INDEX idx_endpoints_endpoint_class ON endpoints(tenant_id, endpoint_class)
    WHERE endpoint_class IS NOT NULL;

-- Basestation receptions: Add composite index for deduplication queries
CREATE INDEX idx_basestation_receptions_dedup ON basestation_receptions(message_id, rssi DESC, snr DESC);

-- Endpoint sessions: Add index for session lookup by short address
CREATE INDEX idx_endpoint_sessions_sh_addr ON endpoint_sessions(tenant_id, sh_addr, status)
    WHERE sh_addr IS NOT NULL AND status = 'active';

-- Downlink queue: Add index for basestation-specific queue queries
CREATE INDEX idx_downlink_queue_basestation ON downlink_queue(transmitted_by_basestation_id, status, priority DESC, created_at)
    WHERE transmitted_by_basestation_id IS NOT NULL AND status IN ('pending', 'scheduled');

-- System events: Add index for endpoint-specific event history
CREATE INDEX idx_system_events_endpoint_history ON system_events(endpoint_id, event_category, occurred_at DESC)
    WHERE endpoint_id IS NOT NULL;

-- Create statistics tracking table for query optimization
CREATE TABLE IF NOT EXISTS table_statistics (
    table_name VARCHAR(100) PRIMARY KEY,
    row_count BIGINT DEFAULT 0,
    last_analyzed TIMESTAMPTZ DEFAULT NOW(),
    avg_row_size INTEGER,
    index_usage JSONB DEFAULT '{}'
);

-- Function to update table statistics
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
                (SELECT COUNT(*) FROM public.messages WHERE false), -- Placeholder, would be dynamic
                NOW())
        ON CONFLICT (table_name) 
        DO UPDATE SET 
            row_count = EXCLUDED.row_count,
            last_analyzed = NOW();
    END LOOP;
END;
$$ LANGUAGE plpgsql;

-- Add table comments for index usage
COMMENT ON INDEX idx_messages_packet_counter IS 'Optimize packet deduplication queries';
COMMENT ON INDEX idx_messages_uplink_mode IS 'Optimize uplink mode filtering';
COMMENT ON INDEX idx_messages_dl_flags IS 'Optimize downlink-ready endpoint queries';
COMMENT ON INDEX idx_endpoints_endpoint_class IS 'Optimize endpoint class filtering';
COMMENT ON INDEX idx_basestation_receptions_dedup IS 'Optimize message deduplication';
COMMENT ON INDEX idx_endpoint_sessions_sh_addr IS 'Optimize short address lookups';
COMMENT ON INDEX idx_downlink_queue_basestation IS 'Optimize basestation-specific queue queries';
COMMENT ON INDEX idx_system_events_endpoint_history IS 'Optimize endpoint-specific event history';
