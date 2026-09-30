-- Drop the delivery outbox and restore the legacy deduplication table in the
-- shape it had before the up migration (018 plus the tenant foreign key
-- 000134 restored), with its updated_at trigger and table comment. The
-- archived rows stay in system_events; the classifier rows written since the
-- up migration are discarded because the legacy shape cannot hold UUID
-- message ids. update_mioty_message_timestamp() is never dropped by the up
-- migration, so the trigger references it directly.

DROP INDEX IF EXISTS idx_message_delivery_outbox_tenant;
DROP INDEX IF EXISTS idx_message_delivery_outbox_due;
DROP TABLE IF EXISTS message_delivery_outbox;
DROP TYPE IF EXISTS message_delivery_channel;

DROP INDEX IF EXISTS idx_mioty_dedup_first_message;
DROP TABLE IF EXISTS mioty_message_deduplication;

CREATE TABLE mioty_message_deduplication (
    id BIGSERIAL PRIMARY KEY,
    tenant_id BIGINT NOT NULL,
    ep_eui BYTEA NOT NULL CHECK (length(ep_eui) = 8),
    packet_cnt BIGINT NOT NULL,
    message_hash BYTEA NOT NULL,
    first_received_at TIMESTAMPTZ NOT NULL,
    first_message_id BIGINT NOT NULL,
    first_basestation_id BIGINT REFERENCES basestations(id) ON DELETE SET NULL,
    duplicate_count INTEGER DEFAULT 0,
    last_duplicate_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT mioty_message_deduplication_tenant_id_fkey FOREIGN KEY (tenant_id) REFERENCES tenants(id) ON DELETE CASCADE,
    CONSTRAINT mioty_dedup_unique UNIQUE(tenant_id, ep_eui, packet_cnt, message_hash)
);

CREATE INDEX idx_mioty_dedup_lookup ON mioty_message_deduplication USING BTREE (ep_eui, packet_cnt);

CREATE TRIGGER trigger_mioty_dedup_updated_at
    BEFORE UPDATE ON mioty_message_deduplication
    FOR EACH ROW
    EXECUTE FUNCTION update_mioty_message_timestamp();

COMMENT ON TABLE mioty_message_deduplication IS 'Message deduplication tracking as performed by Service Center';
