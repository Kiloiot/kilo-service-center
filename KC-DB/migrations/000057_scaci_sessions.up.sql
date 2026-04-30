-- Migration: 000057_scaci_sessions
-- Description: Create tables for SCACI (Service Center to Application Center Interface) session management
-- Implements: MIOTY SCACI v1.0.0 Section 1 (Overview) session persistence requirements
-- Author: KiloCenter Development Team
-- Date: 2025-01-30

-- =============================================================================
-- SCACI Sessions Table
-- =============================================================================
-- Tracks Application Center connections per SCACI §1 session management
-- Requirements:
--   - SCACI-S.1-02: AC establishes connection, SC accepts
--   - SCACI-S.1-05: Session resumption after disconnect requires UUID + opId consistency
--   - SCACI-S.1-04: Mutual TLS authentication required

CREATE TABLE IF NOT EXISTS scaci_sessions (
    id BIGSERIAL PRIMARY KEY,
    tenant_id BIGINT NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,

    -- Application Center identification
    ac_eui BYTEA NOT NULL CHECK (length(ac_eui) = 8),  -- Application Center EUI64

    -- SCACI Session identifiers (SCACI §1 Overview, lines 22-27)
    sn_ac_uuid BYTEA NOT NULL CHECK (length(sn_ac_uuid) = 16), -- AC session UUID
    sn_sc_uuid BYTEA NOT NULL CHECK (length(sn_sc_uuid) = 16), -- SC session UUID

    -- Operation ID tracking for session resumption (SCACI §1, lines 22-27)
    -- AC uses positive IDs (incrementing), SC uses negative IDs (decrementing)
    last_op_id_ac BIGINT NOT NULL DEFAULT 0,  -- Last known AC operation ID (positive)
    last_op_id_sc BIGINT NOT NULL DEFAULT 0,  -- Last known SC operation ID (negative)

    -- Connection state
    status VARCHAR(20) NOT NULL DEFAULT 'active'
        CHECK (status IN ('active', 'resumed', 'disconnected', 'terminated')),

    -- TLS certificate information (SCACI §1, lines 19-20)
    certificate_fingerprint TEXT,              -- SHA256 fingerprint of AC client cert (hex)
    client_cert_subject TEXT,                  -- Certificate DN (e.g., CN=application-center-client)

    -- Network information
    remote_addr TEXT,                          -- Client IP address

    -- Session timing
    connected_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    last_heartbeat TIMESTAMPTZ,                -- Last ping/status received
    disconnected_at TIMESTAMPTZ,               -- When connection was lost

    -- Session resumption control (SCACI §1, lines 22-27)
    can_resume BOOLEAN DEFAULT TRUE,           -- Whether session can be resumed

    -- Extensibility
    metadata JSONB DEFAULT '{}',               -- Additional session metadata

    -- Audit fields
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- =============================================================================
-- SCACI Operation Log Table
-- =============================================================================
-- Tracks SCACI protocol operations for audit and resumption
-- Implements three-way handshake tracking: Request → Response → Complete

CREATE TABLE IF NOT EXISTS scaci_operation_log (
    id BIGSERIAL PRIMARY KEY,
    session_id BIGINT NOT NULL REFERENCES scaci_sessions(id) ON DELETE CASCADE,
    tenant_id BIGINT NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,

    -- Operation identification
    op_id BIGINT NOT NULL,                     -- Signed operation ID (AC:positive, SC:negative)
    command VARCHAR(64) NOT NULL,              -- SCACI command: connect, ulData, dlDataQue, etc.
    direction VARCHAR(10) NOT NULL CHECK (direction IN ('inbound', 'outbound')),

    -- Operation state tracking
    state VARCHAR(20) NOT NULL DEFAULT 'pending'
        CHECK (state IN ('pending', 'acknowledged', 'completed', 'failed')),

    -- Operation data
    request_data JSONB,                        -- Request payload (JSON representation)
    response_data JSONB,                       -- Response payload
    error_message TEXT,                        -- Error details if failed

    -- Timing
    initiated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    acknowledged_at TIMESTAMPTZ,               -- When response received
    completed_at TIMESTAMPTZ,                  -- When complete message received

    -- Audit
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- =============================================================================
-- Indexes
-- =============================================================================

-- SCACI Sessions indexes
CREATE INDEX idx_scaci_sessions_tenant
    ON scaci_sessions(tenant_id, created_at DESC);

CREATE INDEX idx_scaci_sessions_ac_eui
    ON scaci_sessions(ac_eui, tenant_id);

CREATE INDEX idx_scaci_sessions_ac_uuid
    ON scaci_sessions(sn_ac_uuid);

CREATE INDEX idx_scaci_sessions_sc_uuid
    ON scaci_sessions(sn_sc_uuid);

CREATE INDEX idx_scaci_sessions_status
    ON scaci_sessions(status, tenant_id);

-- Ensure only one active session per AC per tenant
CREATE UNIQUE INDEX idx_scaci_sessions_unique_active
    ON scaci_sessions(tenant_id, ac_eui)
    WHERE status = 'active';

-- SCACI Operation Log indexes
CREATE INDEX idx_scaci_op_log_session
    ON scaci_operation_log(session_id, initiated_at DESC);

CREATE INDEX idx_scaci_op_log_tenant
    ON scaci_operation_log(tenant_id, initiated_at DESC);

CREATE INDEX idx_scaci_op_log_op_id
    ON scaci_operation_log(op_id, session_id);

CREATE INDEX idx_scaci_op_log_state
    ON scaci_operation_log(state, session_id)
    WHERE state IN ('pending', 'acknowledged');

CREATE INDEX idx_scaci_op_log_command
    ON scaci_operation_log(command, tenant_id, initiated_at DESC);

-- =============================================================================
-- Triggers for automatic timestamp updates
-- =============================================================================

-- Reuse existing timestamp update function if available, or create it
DO $$
BEGIN
    IF NOT EXISTS (SELECT 1 FROM pg_proc WHERE proname = 'update_updated_at_column') THEN
        CREATE FUNCTION update_updated_at_column()
        RETURNS TRIGGER AS $func$
        BEGIN
            NEW.updated_at = NOW();
            RETURN NEW;
        END;
        $func$ LANGUAGE plpgsql;
    END IF;
END $$;

CREATE TRIGGER trigger_scaci_sessions_updated_at
    BEFORE UPDATE ON scaci_sessions
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER trigger_scaci_op_log_updated_at
    BEFORE UPDATE ON scaci_operation_log
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();

-- =============================================================================
-- Table Comments (Documentation)
-- =============================================================================

COMMENT ON TABLE scaci_sessions IS
    'SCACI (Service Center to Application Center Interface) session management per MIOTY SCACI v1.0.0 §1. '
    'Tracks AC connections, mutual TLS authentication, and session resumption state.';

COMMENT ON COLUMN scaci_sessions.ac_eui IS
    'Application Center EUI64 identifier (8 bytes). Identifies the external AC connecting to this SC.';

COMMENT ON COLUMN scaci_sessions.sn_ac_uuid IS
    'Application Center session UUID (16 bytes). AC provides this on connect; must match for session resumption.';

COMMENT ON COLUMN scaci_sessions.sn_sc_uuid IS
    'Service Center session UUID (16 bytes). SC generates this on connect; must match for session resumption.';

COMMENT ON COLUMN scaci_sessions.last_op_id_ac IS
    'Last known Application Center operation ID (positive, incrementing). Used to validate session resumption per SCACI §1.';

COMMENT ON COLUMN scaci_sessions.last_op_id_sc IS
    'Last known Service Center operation ID (negative, decrementing). Used to validate session resumption per SCACI §1.';

COMMENT ON COLUMN scaci_sessions.can_resume IS
    'Whether this session can be resumed after disconnect. Set to FALSE if inconsistent state detected.';

COMMENT ON TABLE scaci_operation_log IS
    'Audit log of SCACI protocol operations. Tracks three-way handshake completion and operation state.';

COMMENT ON COLUMN scaci_operation_log.op_id IS
    'Operation ID (signed 64-bit). Positive for AC-initiated operations, negative for SC-initiated operations.';

COMMENT ON COLUMN scaci_operation_log.command IS
    'SCACI command name: connect, connectResponse, connectComplete, ping, status, register, ulData, dlDataQue, etc.';

COMMENT ON COLUMN scaci_operation_log.state IS
    'Operation state: pending (request sent), acknowledged (response received), completed (complete msg received), failed (error occurred).';

-- =============================================================================
-- Design Notes
-- =============================================================================
--
-- DOWNLINK QUEUE DECISION:
-- The existing `mioty_messages` table (migration 018) already supports SCACI downlink
-- via `interface_type='scaci'` and `message_type='dl_data'`. The `queue_id` field
-- (added in migration 028) provides the necessary correlation for dlDataQue operations.
-- Therefore, NO separate `scaci_downlink_queue` table is needed.
--
-- SESSION RESUMPTION FLOW:
-- 1. AC reconnects with previous sn_ac_uuid + last known opId
-- 2. SC looks up session by sn_ac_uuid
-- 3. If found and can_resume=true, validate opIds are consistent
-- 4. If consistent, update status to 'resumed' and continue
-- 5. If inconsistent or can_resume=false, start new session
--
-- COMPLIANCE MAPPING:
-- - SCACI-S.1-01: TLS listener implemented separately in KC-Core/pkg/scaci/server.go
-- - SCACI-S.1-02: AC-initiated connections enforced at protocol level
-- - SCACI-S.1-04: Mutual TLS enforced; certificate fingerprint stored here
-- - SCACI-S.1-05: Session resumption logic uses sn_ac_uuid/sn_sc_uuid + opId validation
-- - SCACI-S.1-06: MessagePack framing implemented separately in KC-Core/pkg/scaci/frame.go