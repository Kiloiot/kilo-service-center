-- An Application Center queue id names one in-flight downlink of its
-- organization.
--
-- SCACI §3.10.1 lets the Application Center assign the queId of a dlDataQue,
-- and §3.12.1 reports the downlink's result under it, so the id names the
-- downlink until that result. The index kept the id unique within the tenant
-- for good: an Application Center of one organization blocked the id for
-- every other organization of the tenant (SCACI §1), and no Application
-- Center could use an id again after its downlink ended. The id is now unique
-- among the organization's downlinks still in flight.
--
-- downlink_queue_in_flight declares the in-flight queue states once: the
-- index predicate and every in-flight query of the service center call it. A
-- migration that changes its list rebuilds this index.

CREATE FUNCTION downlink_queue_in_flight(status text) RETURNS boolean
    LANGUAGE sql IMMUTABLE PARALLEL SAFE
    AS $$ SELECT status IN ('pending', 'scheduled', 'reserved', 'queued') $$;

DROP INDEX idx_downlink_queue_tenant_ac_que_id;

CREATE UNIQUE INDEX idx_downlink_queue_org_ac_que_id_in_flight
    ON downlink_queue (tenant_id, organization_id, ac_que_id)
    WHERE ac_que_id IS NOT NULL AND downlink_queue_in_flight(status);
