-- Off for every tenant, so an upgraded installation decodes as before.

ALTER TABLE tenants ADD COLUMN IF NOT EXISTS is_snapshot_only_decoding BOOLEAN NOT NULL DEFAULT FALSE;
