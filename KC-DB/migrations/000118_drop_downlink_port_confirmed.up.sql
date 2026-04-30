-- Remove non-MIOTY fields from downlink_queue (port and confirmed are LoRaWAN artifacts)
ALTER TABLE downlink_queue DROP COLUMN IF EXISTS port;
ALTER TABLE downlink_queue DROP COLUMN IF EXISTS confirmed;
