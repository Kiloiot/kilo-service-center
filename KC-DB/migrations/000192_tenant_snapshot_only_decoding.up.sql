-- A tenant may decode its endpoints only from their pinned blueprint
-- snapshots: an endpoint without a snapshot is not decoded, and creating an
-- endpoint or changing its model ends with a snapshot or is refused. Off for
-- every tenant, so an upgraded installation decodes as before.
--
-- Adding a boolean with a constant default rewrites nothing and holds the
-- tenants lock only for the catalog change.

ALTER TABLE tenants ADD COLUMN IF NOT EXISTS is_snapshot_only_decoding BOOLEAN NOT NULL DEFAULT FALSE;
