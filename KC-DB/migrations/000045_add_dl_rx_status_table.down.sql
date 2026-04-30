-- Remove DL RX Status table and related objects

-- Drop trigger
DROP TRIGGER IF EXISTS trigger_update_dl_rx_status_updated_at ON dl_rx_status;

-- Drop function
DROP FUNCTION IF EXISTS update_dl_rx_status_updated_at();

-- Drop indexes
DROP INDEX IF EXISTS idx_dl_rx_status_endpoint;
DROP INDEX IF EXISTS idx_dl_rx_status_basestation;
DROP INDEX IF EXISTS idx_dl_rx_status_time;

-- Drop table
DROP TABLE IF EXISTS dl_rx_status;