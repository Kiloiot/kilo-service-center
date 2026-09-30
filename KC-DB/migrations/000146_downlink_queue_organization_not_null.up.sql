-- Make downlink_queue.organization_id mandatory. Dispatch derives the delivery
-- organization from the queue row, so an ownerless row is undeliverable and
-- enqueue now rejects a missing organization at the repository layer.
--
-- Rows enqueued before that enforcement are backfilled with the tenant's
-- deterministic default organization (the same rule migration 130 applied:
-- oldest organization per tenant). A row whose tenant has no organization at
-- all cannot be assigned an owner and aborts the migration for operator
-- review: either create the organization or delete the stale queue rows.

WITH default_orgs AS (
    SELECT DISTINCT ON (tenant_id) tenant_id, org_id
    FROM organizations
    ORDER BY tenant_id, created_at ASC, org_id ASC
)
UPDATE downlink_queue dq
SET organization_id = d.org_id
FROM default_orgs d
WHERE d.tenant_id = dq.tenant_id AND dq.organization_id IS NULL;

DO $$
DECLARE
    ownerless_count bigint;
BEGIN
    SELECT count(*) INTO ownerless_count
    FROM downlink_queue
    WHERE organization_id IS NULL;

    IF ownerless_count > 0 THEN
        RAISE EXCEPTION 'downlink_queue has % row(s) whose tenant has no organization; create the organization or delete the stale rows, then re-run', ownerless_count;
    END IF;
END $$;

ALTER TABLE downlink_queue
    ALTER COLUMN organization_id SET NOT NULL;
