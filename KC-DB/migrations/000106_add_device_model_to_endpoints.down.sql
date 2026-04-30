-- Migration 000106 rollback: Remove device_model_id and calibration_data from endpoints

DROP INDEX IF EXISTS idx_endpoints_device_model;

ALTER TABLE endpoints
    DROP COLUMN IF EXISTS calibration_data,
    DROP COLUMN IF EXISTS device_model_id;
