-- Backfill manager permissions for admin users.
-- Admin users should have all manager flags enabled.
UPDATE identity.users
SET is_tenant_manager = true,
    is_base_station_manager = true,
    is_endpoint_manager = true
WHERE is_admin = true
  AND (is_tenant_manager = false OR is_base_station_manager = false OR is_endpoint_manager = false);
