-- Rollback BSSCI §5.7 detach telemetry columns

DROP INDEX IF EXISTS idx_endpoints_last_attached_bs_eui;
DROP INDEX IF EXISTS idx_endpoints_propagate_status;

ALTER TABLE endpoints
DROP COLUMN IF EXISTS propagate_status,
DROP COLUMN IF EXISTS last_detach_packet_cnt,
DROP COLUMN IF EXISTS last_detach_sign,
DROP COLUMN IF EXISTS last_detach_time,
DROP COLUMN IF EXISTS last_propagate_time,
DROP COLUMN IF EXISTS last_attached_bs_eui;
