-- Name the endpoint's decision time for what it records: when the service
-- center last attached the endpoint or changed its ep_status.
--
-- A base station that resumes its session keeps the attachments it held and
-- the downlinks it queued for them (BSSCI §1), and an attach propagate for an
-- endpoint it holds makes it discard those downlinks. A resumed station is
-- therefore sent attPrp only for the endpoints attached after it left (§3.8)
-- and detPrp for those detached meanwhile (§3.9). Every attach decision
-- restates the keys and parameters the stations hold, an over-the-air attach
-- of an endpoint that is already attached included, so its time is recorded
-- even when ep_status stays; a detach records its time only when it changes
-- ep_status. The column and its values stay; only the name and comment change.

ALTER TABLE endpoints RENAME COLUMN status_changed_at TO attachment_changed_at;

COMMENT ON COLUMN endpoints.attachment_changed_at IS
    'When the service center last attached the endpoint or changed its ep_status; NULL before the first recorded change';
