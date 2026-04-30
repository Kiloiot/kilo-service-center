-- BSSCI §5.7 detach telemetry columns (BIGINT nanoseconds, NOT TIMESTAMPTZ)
-- Resolves Finding 2: Missing database columns for detach/attach state tracking

ALTER TABLE endpoints
ADD COLUMN IF NOT EXISTS last_attached_bs_eui BIGINT,
ADD COLUMN IF NOT EXISTS last_propagate_time BIGINT,
ADD COLUMN IF NOT EXISTS last_detach_time BIGINT,
ADD COLUMN IF NOT EXISTS last_detach_sign BYTEA,
ADD COLUMN IF NOT EXISTS last_detach_packet_cnt INTEGER,
ADD COLUMN IF NOT EXISTS propagate_status VARCHAR(50);

COMMENT ON COLUMN endpoints.last_attached_bs_eui IS 'Last BS EUI that handled attach/detach propagate (BSSCI §5.6/5.7)';
COMMENT ON COLUMN endpoints.last_propagate_time IS 'Unix UTC nanoseconds of last propagate operation';
COMMENT ON COLUMN endpoints.last_detach_time IS 'Unix UTC nanoseconds of last detach reception (BSSCI §5.7.1)';
COMMENT ON COLUMN endpoints.last_detach_sign IS '4-byte endpoint signature from detach message';
COMMENT ON COLUMN endpoints.last_detach_packet_cnt IS 'Endpoint packet counter from detach message';
COMMENT ON COLUMN endpoints.propagate_status IS 'Propagation state: pending, completed, failed';

CREATE INDEX IF NOT EXISTS idx_endpoints_propagate_status
    ON endpoints(propagate_status) WHERE propagate_status IS NOT NULL;
CREATE INDEX IF NOT EXISTS idx_endpoints_last_attached_bs_eui
    ON endpoints(last_attached_bs_eui) WHERE last_attached_bs_eui IS NOT NULL;
