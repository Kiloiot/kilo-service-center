-- Add role flag columns for granular user permissions
-- These columns support the user admin UI without modifying auth-only UserStore

ALTER TABLE users ADD COLUMN is_tenant_manager BOOLEAN NOT NULL DEFAULT FALSE;
ALTER TABLE users ADD COLUMN is_base_station_manager BOOLEAN NOT NULL DEFAULT FALSE;
ALTER TABLE users ADD COLUMN is_endpoint_manager BOOLEAN NOT NULL DEFAULT FALSE;

COMMENT ON COLUMN users.is_tenant_manager IS 'Can manage tenant configuration and settings';
COMMENT ON COLUMN users.is_base_station_manager IS 'Can manage base station registrations and configuration';
COMMENT ON COLUMN users.is_endpoint_manager IS 'Can manage endpoint registrations and operations';
