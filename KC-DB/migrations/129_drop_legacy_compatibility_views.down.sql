-- Re-create legacy compatibility views for rollback

CREATE OR REPLACE VIEW gateways AS
SELECT
    id,
    encode(bs_eui, 'hex') as gateway_eui,
    tenant_id,
    name,
    description,
    latitude,
    longitude,
    altitude,
    CASE WHEN is_online THEN 'online' ELSE 'offline' END as status,
    tags,
    created_at,
    updated_at,
    last_seen_at
FROM basestations;

CREATE OR REPLACE VIEW devices AS
SELECT
    id,
    encode(ep_eui, 'hex') as device_eui,
    tenant_id,
    name,
    description,
    endpoint_class as device_class,
    nwk_key as network_key,
    app_key,
    ep_status as status,
    tags,
    created_at,
    updated_at,
    last_seen_at
FROM endpoints;

COMMENT ON VIEW gateways IS 'Legacy compatibility view mapping basestations to gateways (deprecated - use basestations directly)';
COMMENT ON VIEW devices IS 'Legacy compatibility view mapping endpoints to devices (deprecated - use endpoints directly)';
