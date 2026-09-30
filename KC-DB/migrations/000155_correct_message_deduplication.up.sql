-- Make PostgreSQL the only uplink classifier.
--
-- The legacy mioty_message_deduplication table (018) was written by the
-- EUI-rename sweep only; classification ran in KC-Core memory, so two
-- processes or a restart could admit the same packet twice. Its rows carry
-- BIGINT first_message_id values that referred to the pre-000047 messages
-- table and match nothing today, so they are archived to tenant-scoped
-- system_events with count parity and never cast to UUID.
--
-- The table is recreated with a static unique key on
-- (owner_tenant_id, ep_eui, packet_cnt): the duplicate window is applied by
-- the writer against first_received_at because an index predicate cannot
-- reference now(). message_hash is comparison data, not part of the key.
-- first_message_id is a deferrable foreign key so the classifier row and
-- the message it points to are written in one transaction in either order.
--
-- message_delivery_outbox queues every new message once per channel; a
-- worker claims due rows with SKIP LOCKED, so fan-out survives a crash and
-- a slow Application Center cannot lose messages.

DO $$
DECLARE
    source_count bigint;
    archived_count bigint;
BEGIN
    SELECT count(*) INTO source_count FROM mioty_message_deduplication;

    WITH archived AS (
        INSERT INTO system_events (
            tenant_id, event_type, event_category, severity,
            source_type, source_name, title, description,
            data, occurred_at, recorded_at, status
        )
        SELECT
            d.tenant_id,
            'message.dedup.archived',
            'message',
            'info',
            'migration',
            encode(d.ep_eui, 'hex'),
            format('Archived legacy deduplication row %s', d.id),
            format('Legacy mioty_message_deduplication row %s archived by migration 000155', d.id),
            jsonb_build_object(
                'legacyTable', 'mioty_message_deduplication',
                'legacyId', d.id,
                'epEui', encode(d.ep_eui, 'hex'),
                'packetCnt', d.packet_cnt,
                'messageHash', encode(d.message_hash, 'hex'),
                'firstReceivedAt', d.first_received_at,
                'legacyFirstMessageId', d.first_message_id,
                'firstBasestationId', d.first_basestation_id,
                'duplicateCount', d.duplicate_count,
                'lastDuplicateAt', d.last_duplicate_at,
                'createdAt', d.created_at
            ),
            COALESCE(d.first_received_at, d.created_at, NOW()),
            NOW(),
            'resolved'
        FROM mioty_message_deduplication d
        RETURNING 1
    )
    SELECT count(*) INTO archived_count FROM archived;

    IF archived_count <> source_count THEN
        RAISE EXCEPTION 'mioty_message_deduplication archive mismatch: % source row(s), % archived', source_count, archived_count;
    END IF;
END $$;

DROP INDEX IF EXISTS idx_mioty_dedup_lookup;
DROP TABLE IF EXISTS mioty_message_deduplication;

CREATE TABLE mioty_message_deduplication (
    owner_tenant_id   BIGINT NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    ep_eui            BYTEA NOT NULL CHECK (length(ep_eui) = 8),
    packet_cnt        BIGINT NOT NULL,
    message_hash      BYTEA NOT NULL,
    first_message_id  UUID NOT NULL REFERENCES messages(id) ON DELETE CASCADE DEFERRABLE INITIALLY DEFERRED,
    first_bs_eui      BYTEA NOT NULL CHECK (length(first_bs_eui) = 8),
    first_received_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    last_received_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    duplicate_count   INTEGER NOT NULL DEFAULT 0,
    PRIMARY KEY (owner_tenant_id, ep_eui, packet_cnt)
);

CREATE INDEX idx_mioty_dedup_first_message ON mioty_message_deduplication(first_message_id);

CREATE TYPE message_delivery_channel AS ENUM ('scaci', 'mqtt');

CREATE TABLE message_delivery_outbox (
    message_id      UUID NOT NULL REFERENCES messages(id) ON DELETE CASCADE,
    channel         message_delivery_channel NOT NULL,
    owner_tenant_id BIGINT NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    status          VARCHAR(20) NOT NULL DEFAULT 'pending' CHECK (status IN ('pending', 'delivered', 'parked')),
    attempts        INTEGER NOT NULL DEFAULT 0,
    next_attempt_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    last_error      TEXT,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    delivered_at    TIMESTAMPTZ,
    PRIMARY KEY (message_id, channel)
);

CREATE INDEX idx_message_delivery_outbox_due ON message_delivery_outbox(next_attempt_at)
    WHERE status = 'pending';
CREATE INDEX idx_message_delivery_outbox_tenant ON message_delivery_outbox(owner_tenant_id, created_at DESC);
