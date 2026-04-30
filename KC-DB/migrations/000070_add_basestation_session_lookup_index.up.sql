-- Migration 000070: Add index for efficient basestation_sessions lookup
-- BSSCI §5-5.3 compliance: Support fast session metadata retrieval for offline stations
-- This index supports the LATERAL join pattern for fetching the most recent session per base station
-- Used by: KC-DB/storage/queries/operation_status_queries.go SQLBSSCIStatusSummary

CREATE INDEX IF NOT EXISTS idx_basestation_sessions_bs_lookup
ON basestation_sessions(basestation_id, status, started_at DESC);

-- Index rationale:
-- 1. basestation_id: Filter sessions for specific base station
-- 2. status: Filter to 'active' or 'resumed' sessions only
-- 3. started_at DESC: Order by most recent first (LIMIT 1 in LATERAL join)
--
-- Performance: Enables index-only scan for session metadata queries
-- without scanning full basestation_sessions table
