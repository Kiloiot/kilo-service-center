-- Re-add deprecated columns for rollback
ALTER TABLE basestations ADD COLUMN IF NOT EXISTS host VARCHAR(255);
ALTER TABLE basestations ADD COLUMN IF NOT EXISTS port INTEGER;
ALTER TABLE basestations ADD COLUMN IF NOT EXISTS tls_enabled BOOLEAN DEFAULT false;
