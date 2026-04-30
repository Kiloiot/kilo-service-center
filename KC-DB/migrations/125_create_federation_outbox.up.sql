-- Federation outbox: durable queue of uplinks waiting to be relayed from CE to ECE.
CREATE TABLE IF NOT EXISTS federation_outbox (
    relay_id       UUID PRIMARY KEY,
    ep_eui         BIGINT NOT NULL,
    bs_eui         BIGINT NOT NULL,
    raw_frame      BYTEA NOT NULL,
    received_at_ns BIGINT NOT NULL,
    status         VARCHAR(50) NOT NULL DEFAULT 'pending',
    last_error     TEXT,
    sent_at        TIMESTAMPTZ,
    acked_at       TIMESTAMPTZ,
    created_at     TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_federation_outbox_status ON federation_outbox(status, created_at);
