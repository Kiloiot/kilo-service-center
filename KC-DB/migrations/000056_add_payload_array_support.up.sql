-- Migration: Add support for counter-dependent payload arrays per MIOTY BSSCI §3.12.1
-- When cntDepend=true, multiple payloads can be provided with corresponding packet counters.
-- We store the full canonical MIOTY DLDataQueue structure to preserve all payloads.

-- Add user_data column to store the canonical MIOTY DLDataQueue
ALTER TABLE downlink_queue
  ADD COLUMN user_data JSONB;

-- Add comment documenting the purpose and structure
COMMENT ON COLUMN downlink_queue.user_data IS
  'Canonical MIOTY DLDataQueue with UserData [][]byte for counter-dependent messages (BSSCI §3.12.1)';

-- The existing 'payload' column remains for backward compatibility and stores the first payload