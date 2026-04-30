-- Rollback migration 000070: Remove basestation_sessions lookup index

DROP INDEX IF EXISTS idx_basestation_sessions_bs_lookup;
