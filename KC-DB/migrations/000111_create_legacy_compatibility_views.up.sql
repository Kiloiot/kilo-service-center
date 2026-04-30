-- Migration 000111: Create legacy compatibility views
-- These views provide backward compatibility for legacy code that uses
-- 'gateways' and 'devices' table names instead of 'basestations' and 'endpoints'

-- gateways view maps to basestations with legacy column names
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

-- devices view maps to endpoints with legacy column names
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

-- Add comments for documentation
COMMENT ON VIEW gateways IS 'Legacy compatibility view mapping basestations to gateways (deprecated - use basestations directly)';
COMMENT ON VIEW devices IS 'Legacy compatibility view mapping endpoints to devices (deprecated - use endpoints directly)';
