-- Add MIOTY Base Station Status Response Fields per BSSCI v1.0.0 Section 3.5.2
-- This migration adds the missing statusRsp fields from the MIOTY specification

-- Add status response fields to basestations table per MIOTY BSSCI 3.5.2
ALTER TABLE basestations
ADD COLUMN IF NOT EXISTS system_time BIGINT,
ADD COLUMN IF NOT EXISTS duty_cycle FLOAT,
ADD COLUMN IF NOT EXISTS uptime_seconds BIGINT,
ADD COLUMN IF NOT EXISTS temperature_celsius FLOAT,
ADD COLUMN IF NOT EXISTS cpu_load FLOAT,
ADD COLUMN IF NOT EXISTS memory_load FLOAT,
ADD COLUMN IF NOT EXISTS bs_config JSONB,
ADD COLUMN IF NOT EXISTS last_status_at TIMESTAMPTZ;

-- Add comments for MIOTY compliance documentation
COMMENT ON COLUMN basestations.system_time IS 'Unix UTC system time from base station, 64 bit, ns resolution per MIOTY BSSCI 3.5.2';
COMMENT ON COLUMN basestations.duty_cycle IS 'Fraction of TX time, sliding window over one hour per MIOTY BSSCI 3.5.2';
COMMENT ON COLUMN basestations.uptime_seconds IS 'System uptime in seconds from base station per MIOTY BSSCI 3.5.2';
COMMENT ON COLUMN basestations.temperature_celsius IS 'System temperature in degree Celsius from base station per MIOTY BSSCI 3.5.2';
COMMENT ON COLUMN basestations.cpu_load IS 'CPU utilization, normalized to 1.0 for all cores per MIOTY BSSCI 3.5.2';
COMMENT ON COLUMN basestations.memory_load IS 'Memory utilization, normalized to 1.0 per MIOTY BSSCI 3.5.2';
COMMENT ON COLUMN basestations.bs_config IS 'Configuration object from base station per MIOTY BSSCI 3.5.2';
COMMENT ON COLUMN basestations.last_status_at IS 'Timestamp when status response was last received from base station';

-- Add indexes for efficient status queries
CREATE INDEX IF NOT EXISTS idx_basestations_last_status 
ON basestations(tenant_id, last_status_at DESC) 
WHERE last_status_at IS NOT NULL;

CREATE INDEX IF NOT EXISTS idx_basestations_system_health 
ON basestations(tenant_id, temperature_celsius, cpu_load, memory_load) 
WHERE temperature_celsius IS NOT NULL OR cpu_load IS NOT NULL OR memory_load IS NOT NULL;