-- Restore one Application Center queue id per tenant for good. The guard
-- aborts while an id repeats within a tenant, across its organizations or
-- after a downlink ended, which the restored index would reject.

DO $$
DECLARE
    repeated bigint;
BEGIN
    SELECT count(*) INTO repeated FROM (
        SELECT 1 FROM downlink_queue WHERE ac_que_id IS NOT NULL
        GROUP BY tenant_id, ac_que_id HAVING count(*) > 1
    ) ids;
    IF repeated > 0 THEN
        RAISE EXCEPTION 'downlink_queue has % Application Center queue id(s) repeated within a tenant', repeated;
    END IF;
END $$;

DROP INDEX idx_downlink_queue_org_ac_que_id_in_flight;

CREATE UNIQUE INDEX idx_downlink_queue_tenant_ac_que_id
    ON downlink_queue (tenant_id, ac_que_id)
    WHERE ac_que_id IS NOT NULL;

DROP FUNCTION downlink_queue_in_flight(text);
