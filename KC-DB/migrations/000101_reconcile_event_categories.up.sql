-- Migration 100: Reconcile event categories
-- Add categories that code currently uses but DB constraint doesn't allow
-- This brings the DB constraint in sync with actual usage in system_events.go

ALTER TABLE system_events
DROP CONSTRAINT IF EXISTS system_events_event_category_check;

ALTER TABLE system_events
ADD CONSTRAINT system_events_event_category_check
CHECK (event_category IN (
    -- Existing DB categories
    'security',     -- Auth failures, access violations
    'endpoint',     -- Endpoint CRUD, attach/detach
    'basestation',  -- BS CRUD, connect/disconnect
    'message',      -- UL/DL message traffic (EXCLUDE from dashboard)
    'system',       -- Service lifecycle, background jobs
    'roaming',      -- Cross-tenant roaming events
    'error',        -- System errors
    'audit',        -- Admin actions (user/org CRUD goes here)
    -- Categories used by code but missing from DB
    'protocol',     -- Generic BSSCI/SCACI protocol events
    'scaci',        -- SCACI-specific events
    'bssci',        -- BSSCI-specific events
    'session'       -- Protocol sessions, WebSocket connections
));

COMMENT ON COLUMN system_events.event_category IS 'Event category: security, endpoint, basestation, message, system, roaming, error, audit, protocol, scaci, bssci, session';
