-- Migration 000090 rollback: Remove organization_id from downlink_queue
DROP INDEX IF EXISTS idx_downlink_queue_organization_id;
ALTER TABLE downlink_queue DROP COLUMN IF EXISTS organization_id;
