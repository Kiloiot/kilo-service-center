-- A base station that held a downlink may report it sent after the service
-- center reported it expired, for example when it transmitted the downlink
-- while its revoke was under way and its answer crossed the expiry. The
-- expiry stands and the contradiction is filed in the owner's events.
-- sent_after_expiry_at is when the station holding the downlink first
-- reported it so; only that report is filed, so a station repeating its
-- result cannot fill the events. It is NULL for every other row.
--
-- Adding a nullable column without a default rewrites nothing and holds the
-- downlink_queue lock only for the catalog change.

ALTER TABLE downlink_queue ADD COLUMN IF NOT EXISTS sent_after_expiry_at TIMESTAMPTZ;
