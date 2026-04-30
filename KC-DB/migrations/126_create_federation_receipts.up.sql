-- Federation receipts: ECE-side idempotency log for relayed uplinks.
CREATE TABLE IF NOT EXISTS federation_receipts (
    relay_id        UUID PRIMARY KEY,
    ce_id           UUID NOT NULL REFERENCES ce_registry(ce_id),
    ep_eui          BIGINT NOT NULL,
    ce_bs_eui       BIGINT NOT NULL,
    message_id      UUID,
    accepted        BOOLEAN NOT NULL,
    error_message   TEXT,
    owner_tenant_id BIGINT,
    received_at     TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_federation_receipts_ce_id ON federation_receipts(ce_id, received_at);
