-- CE registry: ECE-side table tracking connected Community Edition instances.
CREATE TABLE IF NOT EXISTS ce_registry (
    ce_id                    UUID PRIMARY KEY,
    company_name             VARCHAR(255) NOT NULL,
    status                   VARCHAR(50) NOT NULL DEFAULT 'active',
    connection_status        VARCHAR(50) NOT NULL DEFAULT 'offline',
    token_hash               VARCHAR(64) NOT NULL,
    token_issued_at          TIMESTAMPTZ NOT NULL,
    revoked_at               TIMESTAMPTZ,
    revocation_reason        TEXT,
    first_seen_at            TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    last_connected_at        TIMESTAMPTZ,
    last_heartbeat_at        TIMESTAMPTZ,
    active_basestation_count INTEGER NOT NULL DEFAULT 0,
    total_packets_relayed    BIGINT NOT NULL DEFAULT 0,
    ce_version               VARCHAR(100),
    created_at               TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at               TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_ce_registry_status ON ce_registry(status);
CREATE INDEX IF NOT EXISTS idx_ce_registry_connection_status ON ce_registry(connection_status);
