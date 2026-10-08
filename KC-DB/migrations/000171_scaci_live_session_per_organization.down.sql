-- Restore one live SCACI session per tenant and acEui. The guard aborts while
-- an acEui is live in more than one organization of a tenant, which the
-- restored index would reject.

DO $$
DECLARE
    shared_acs bigint;
BEGIN
    SELECT count(*) INTO shared_acs FROM (
        SELECT 1 FROM scaci_sessions WHERE status = 'active'
        GROUP BY tenant_id, ac_eui HAVING count(*) > 1
    ) live;
    IF shared_acs > 0 THEN
        RAISE EXCEPTION 'scaci_sessions has % acEui(s) live in more than one organization', shared_acs;
    END IF;
END $$;

DROP INDEX idx_scaci_sessions_unique_active;

CREATE UNIQUE INDEX idx_scaci_sessions_unique_active
    ON scaci_sessions (tenant_id, ac_eui)
    WHERE status = 'active';
