-- Record when the service center last changed an endpoint's attachment status.
--
-- A base station that resumes its session kept the endpoints it held (BSSCI
-- §1), so it is sent a detach propagate for every endpoint detached while it
-- was away (§3.9). That needs the time of the detach decision.
-- last_detach_time is the reception time of an over-the-air detach (§3.7.1)
-- and is not written by a service center decision. status_changed_at is
-- written only when ep_status actually changes. For a detached endpoint that
-- predates the column, its last detach time is the closest known decision
-- time; every other endpoint starts unknown.

ALTER TABLE endpoints ADD COLUMN status_changed_at TIMESTAMPTZ;

UPDATE endpoints
SET status_changed_at = to_timestamp(last_detach_time / 1000000000.0)
WHERE ep_status = 'detached' AND last_detach_time IS NOT NULL;

COMMENT ON COLUMN endpoints.status_changed_at IS
    'When the service center last changed ep_status; NULL before the first recorded change';
