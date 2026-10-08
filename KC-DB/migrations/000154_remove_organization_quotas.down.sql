-- Restore the quota columns with the defaults migrations 000099 and 001
-- declared, recreating the public.organizations view so it exposes them
-- again. Values discarded by the up migration are not recoverable; every
-- organization comes back with base stations allowed and unlimited counts,
-- every tenant with the original default limits.
--
-- The restored columns are appended to identity.organizations, so the view
-- names its columns in the order it had before the up migration instead of
-- selecting * in the new physical order.

DROP VIEW IF EXISTS public.organizations;

ALTER TABLE identity.organizations
    ADD COLUMN IF NOT EXISTS can_have_base_stations BOOLEAN NOT NULL DEFAULT true,
    ADD COLUMN IF NOT EXISTS max_base_station_count INTEGER,
    ADD COLUMN IF NOT EXISTS max_endpoint_count INTEGER;

COMMENT ON COLUMN identity.organizations.can_have_base_stations IS 'Whether org can own base stations';
COMMENT ON COLUMN identity.organizations.max_base_station_count IS 'NULL = unlimited';
COMMENT ON COLUMN identity.organizations.max_endpoint_count IS 'NULL = unlimited';

CREATE VIEW public.organizations AS
SELECT org_id, tenant_id, name, state, created_at, updated_at, external_id, description,
       can_have_base_stations, max_base_station_count, max_endpoint_count, tags
FROM identity.organizations;

ALTER TABLE tenants
    ADD COLUMN IF NOT EXISTS max_endpoints INTEGER DEFAULT 1000,
    ADD COLUMN IF NOT EXISTS max_basestations INTEGER DEFAULT 100;
