-- The previous release knows no 'revoking': a downlink being revoked goes
-- back to queued at its holder, where that release's sweep expires it and
-- revokes it again.

UPDATE downlink_queue SET status = 'queued' WHERE status = 'revoking';

DROP INDEX IF EXISTS idx_downlink_queue_revoking_station;
ALTER TABLE downlink_queue DROP COLUMN IF EXISTS revoke_asked_at;
DROP INDEX idx_downlink_queue_org_ac_que_id_in_flight;

CREATE OR REPLACE FUNCTION downlink_queue_in_flight(status text) RETURNS boolean
    LANGUAGE sql IMMUTABLE PARALLEL SAFE
    AS $$ SELECT status IN ('pending', 'scheduled', 'reserved', 'queued') $$;

CREATE UNIQUE INDEX idx_downlink_queue_org_ac_que_id_in_flight
    ON downlink_queue (tenant_id, organization_id, ac_que_id)
    WHERE ac_que_id IS NOT NULL AND downlink_queue_in_flight(status);

ALTER TABLE downlink_queue DROP CONSTRAINT IF EXISTS downlink_queue_status_check;
ALTER TABLE downlink_queue ADD CONSTRAINT downlink_queue_status_check CHECK (status IN (
    'pending', 'scheduled', 'reserved', 'queued', 'transmitted', 'confirmed',
    'failed', 'expired', 'cancelled', 'revoked', 'acked', 'delivered'
));
