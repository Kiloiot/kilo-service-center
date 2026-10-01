-- A downlink a base station holds when its lifetime ends is being revoked.
--
-- Only a dlDataRes "sent" proves a downlink was transmitted (BSSCI §3.14.1);
-- dlDataRevRsp carries no status (§3.13.2). The expiry sweep used to end a
-- held downlink expired before its station answered the dlDataRev, so a
-- transmission the station reported meanwhile was dropped and the
-- originators were told "expired" for a downlink the endpoint received. A
-- held downlink past its lifetime now becomes 'revoking': it stays in flight,
-- keeps its holder, and ends expired only when that station confirms the
-- revoke, answers that it does not hold it, or starts a new session that
-- discarded it (§1); a dlDataRes it reports first ends it with that result.
--
-- downlink_queue_in_flight declares 'revoking' in flight, so the
-- Application Center queue id stays taken until the downlink ends; the
-- index that depends on the function is rebuilt around the new definition.
-- idx_downlink_queue_revoking_station finds the downlinks a reconnecting
-- station is asked to drop. revoke_asked_at is when the holder was last asked
-- to drop a revoking downlink, so the sweep asks a connected holder that has
-- not answered again once per sweep interval; it is NULL for every other row.
--
-- Locking: the CHECK constraint scans downlink_queue once; both index builds
-- hold a SHARE lock, so downlink writes wait until they are built. No row is
-- 'revoking' before this migration, so the rebuilt unique index cannot fail.

ALTER TABLE downlink_queue DROP CONSTRAINT IF EXISTS downlink_queue_status_check;
ALTER TABLE downlink_queue ADD CONSTRAINT downlink_queue_status_check CHECK (status IN (
    'pending', 'scheduled', 'reserved', 'queued', 'revoking', 'transmitted', 'confirmed',
    'failed', 'expired', 'cancelled', 'revoked', 'acked', 'delivered'
));

DROP INDEX idx_downlink_queue_org_ac_que_id_in_flight;

CREATE OR REPLACE FUNCTION downlink_queue_in_flight(status text) RETURNS boolean
    LANGUAGE sql IMMUTABLE PARALLEL SAFE
    AS $$ SELECT status IN ('pending', 'scheduled', 'reserved', 'queued', 'revoking') $$;

CREATE UNIQUE INDEX idx_downlink_queue_org_ac_que_id_in_flight
    ON downlink_queue (tenant_id, organization_id, ac_que_id)
    WHERE ac_que_id IS NOT NULL AND downlink_queue_in_flight(status);

ALTER TABLE downlink_queue ADD COLUMN IF NOT EXISTS revoke_asked_at TIMESTAMPTZ;

CREATE INDEX idx_downlink_queue_revoking_station
    ON downlink_queue (bs_eui)
    WHERE status = 'revoking';
