-- The previous release decodes every endpoint without a snapshot from the catalog.

ALTER TABLE tenants DROP COLUMN IF EXISTS is_snapshot_only_decoding;
