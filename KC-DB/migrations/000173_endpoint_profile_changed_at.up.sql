-- Record when an edit last changed an endpoint's station profile: the attach
-- propagate parameters base stations hold (BSSCI §3.8.1). A base station keeps
-- the profile it was last sent until the endpoint is attached again, so while
-- this time is later than the last completed attach propagate (propagated_at)
-- the stations still hold the earlier profile. Every endpoint starts unknown.

ALTER TABLE endpoints ADD COLUMN profile_changed_at TIMESTAMPTZ;

COMMENT ON COLUMN endpoints.profile_changed_at IS
    'When an edit last changed the attach propagate parameters (BSSCI §3.8.1); NULL when none has';
