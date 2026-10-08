-- Recreate mioty_subpackets in the shape it had before the up migration: the
-- 018 definition minus the composite foreign key to mioty_messages, which
-- 000047 already removed, and with the tenant foreign key 000134 restored.

CREATE TABLE IF NOT EXISTS mioty_subpackets (
    id BIGSERIAL PRIMARY KEY,
    message_id BIGINT NOT NULL,
    message_received_at TIMESTAMPTZ NOT NULL,
    tenant_id BIGINT NOT NULL,
    subpacket_index INTEGER NOT NULL,
    snr REAL,
    rssi REAL,
    frequency BIGINT,
    phase REAL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT mioty_subpackets_tenant_id_fkey FOREIGN KEY (tenant_id) REFERENCES tenants(id) ON DELETE CASCADE,
    CONSTRAINT mioty_subpackets_unique UNIQUE(message_id, message_received_at, subpacket_index, tenant_id)
);

CREATE INDEX IF NOT EXISTS idx_mioty_subpackets_message ON mioty_subpackets USING BTREE (message_id, message_received_at);
CREATE INDEX IF NOT EXISTS idx_mioty_subpackets_tenant ON mioty_subpackets USING BTREE (tenant_id, created_at DESC);
