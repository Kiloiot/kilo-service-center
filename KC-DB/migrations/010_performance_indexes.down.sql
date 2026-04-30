-- Rollback MIOTY performance indexes

-- Drop function and statistics table
DROP FUNCTION IF EXISTS update_table_statistics();
DROP TABLE IF EXISTS table_statistics;

-- Drop all performance indexes
DROP INDEX IF EXISTS idx_messages_packet_counter;
DROP INDEX IF EXISTS idx_messages_uplink_mode;
DROP INDEX IF EXISTS idx_messages_dl_flags;
DROP INDEX IF EXISTS idx_messages_roaming;
DROP INDEX IF EXISTS idx_devices_device_class;
DROP INDEX IF EXISTS idx_devices_attachment;
DROP INDEX IF EXISTS idx_devices_short_addr;
DROP INDEX IF EXISTS idx_gateway_receptions_dedup;
DROP INDEX IF EXISTS idx_device_sessions_sh_addr;
DROP INDEX IF EXISTS idx_downlink_queue_gateway;
DROP INDEX IF EXISTS idx_system_events_device_history;
