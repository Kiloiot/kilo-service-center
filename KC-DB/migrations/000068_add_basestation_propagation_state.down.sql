-- Migration 068 rollback: Remove basestation propagation state table
-- BSSCI §5.8-5.8.3: Automatic attach/detach propagation

DROP TABLE IF EXISTS basestation_propagation_state;
