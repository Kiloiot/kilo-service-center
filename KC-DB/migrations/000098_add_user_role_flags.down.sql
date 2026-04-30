-- Rollback role flag columns

ALTER TABLE users DROP COLUMN IF EXISTS is_endpoint_manager;
ALTER TABLE users DROP COLUMN IF EXISTS is_base_station_manager;
ALTER TABLE users DROP COLUMN IF EXISTS is_tenant_manager;
