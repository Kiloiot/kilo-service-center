-- One live SCACI session per Application Center of an organization.
--
-- An Application Center is its acEui within one organization of a tenant
-- (SCACI §1); sessions without an organization count as one organization.
-- The index allowed one live session per tenant and acEui, so an Application
-- Center of another organization of the tenant that claimed the acEui while
-- the owner was offline locked the owner out until it disconnected.

DROP INDEX idx_scaci_sessions_unique_active;

CREATE UNIQUE INDEX idx_scaci_sessions_unique_active
    ON scaci_sessions (tenant_id, organization_id, ac_eui) NULLS NOT DISTINCT
    WHERE status = 'active';
