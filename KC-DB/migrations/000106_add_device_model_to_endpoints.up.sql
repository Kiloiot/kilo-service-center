-- Migration 000106: Add device_model_id and calibration_data to endpoints table
-- Part of Blueprint feature (MIOTY Application Layer payload decoding)

ALTER TABLE endpoints
    ADD COLUMN IF NOT EXISTS device_model_id UUID REFERENCES device_models(id),
    ADD COLUMN IF NOT EXISTS calibration_data JSONB;

-- Index for device model lookups
CREATE INDEX idx_endpoints_device_model ON endpoints(device_model_id)
    WHERE device_model_id IS NOT NULL;

-- Comments for documentation
COMMENT ON COLUMN endpoints.device_model_id IS 'Reference to device model for blueprint-based decoding';
COMMENT ON COLUMN endpoints.calibration_data IS 'Per-device calibration overrides for blueprint decoding (MIOTY spec §2.2.13)';
