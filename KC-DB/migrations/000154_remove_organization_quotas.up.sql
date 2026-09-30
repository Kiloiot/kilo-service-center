-- Remove the organization and tenant quota columns.
--
-- can_have_base_stations, max_base_station_count and max_endpoint_count
-- (organizations, 000099) and max_basestations, max_endpoints (tenants, 001)
-- were published through the API and shown in the UI but never enforced by
-- any registration or attach path: quotas are a SaaS entitlement concern,
-- not a MIOTY service center function, and the platform above the service
-- center owns them. The columns are dropped and the API fields are reserved.
--
-- public.organizations is the SELECT * view over identity.organizations
-- (000116); a view pins its column list, so it is recreated around the drop.
--
-- Historical quota values are discarded on purpose; the down migration
-- restores the columns with their former defaults but cannot reconstruct
-- the values.

DROP VIEW IF EXISTS public.organizations;

ALTER TABLE identity.organizations
    DROP COLUMN IF EXISTS can_have_base_stations,
    DROP COLUMN IF EXISTS max_base_station_count,
    DROP COLUMN IF EXISTS max_endpoint_count;

CREATE VIEW public.organizations AS SELECT * FROM identity.organizations;

ALTER TABLE tenants
    DROP COLUMN IF EXISTS max_basestations,
    DROP COLUMN IF EXISTS max_endpoints;
