-- Remove MIOTY-specific downlink queue fields

-- Drop indexes
DROP INDEX IF EXISTS idx_downlink_queue_que_id;
DROP INDEX IF EXISTS idx_downlink_queue_pending_ep;

-- Remove MIOTY-specific columns
ALTER TABLE downlink_queue
DROP COLUMN IF EXISTS que_id,
DROP COLUMN IF EXISTS cnt_depend,
DROP COLUMN IF EXISTS packet_cnt,
DROP COLUMN IF EXISTS format,
DROP COLUMN IF EXISTS response_exp,
DROP COLUMN IF EXISTS response_prio,
DROP COLUMN IF EXISTS dl_wind_req,
DROP COLUMN IF EXISTS exp_only,
DROP COLUMN IF EXISTS result,
DROP COLUMN IF EXISTS tx_time;