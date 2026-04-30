-- Add attach subpacket telemetry persistence (BSSCI §5.6.1)
ALTER TABLE endpoints
ADD COLUMN last_attach_subpackets JSONB DEFAULT NULL;

COMMENT ON COLUMN endpoints.last_attach_subpackets IS
  'Last attach subpacket telemetry per BSSCI §5.6.1 (optional field)';
