-- Allow NULL organization_id on downlink_queue again. Backfilled values from
-- the up migration are left in place; they were derived from the tenant's
-- deterministic default organization and remain valid owners.

ALTER TABLE downlink_queue
    ALTER COLUMN organization_id DROP NOT NULL;
