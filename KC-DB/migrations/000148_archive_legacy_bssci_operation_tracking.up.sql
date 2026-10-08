-- Retire the legacy bssci_operation_tracking table (028).
--
-- Session resumption reads bssci_pending_operations (000049/000050), whose
-- rows carry the complete operation_data needed to reissue an operation. The
-- legacy table never stored that payload, so its rows can neither be copied
-- into the canonical table nor completed with fabricated data. Every legacy
-- row is instead archived to a tenant-scoped system_events record (state,
-- command, direction, timing, response and error metadata), the tenant and
-- base station resolved through basestation_sessions, and the table is
-- dropped once the archive count matches the source count.
--
-- A legacy row still marked pending is only acceptable when the canonical
-- table already holds the same operation; otherwise the migration aborts so
-- the operator can inspect the session before the record disappears.

DO $$
DECLARE
    unresumable_count bigint;
    source_count bigint;
    archived_count bigint;
BEGIN
    SELECT count(*) INTO unresumable_count
    FROM bssci_operation_tracking t
    WHERE t.state = 'pending'
      AND NOT EXISTS (
          SELECT 1
          FROM bssci_pending_operations p
          WHERE p.basestation_session_id = t.session_id
            AND p.operation_id = t.op_id
      );

    IF unresumable_count > 0 THEN
        RAISE EXCEPTION 'bssci_operation_tracking has % pending row(s) with no canonical bssci_pending_operations row; resolve them before re-running', unresumable_count;
    END IF;

    SELECT count(*) INTO source_count FROM bssci_operation_tracking;

    WITH archived AS (
        INSERT INTO system_events (
            tenant_id, event_type, event_category, severity,
            source_type, source_name, title, description,
            basestation_id, data, occurred_at, recorded_at, status
        )
        SELECT
            s.tenant_id,
            'bssci.operation.archived',
            'bssci',
            CASE t.state WHEN 'failed' THEN 'warning' ELSE 'info' END,
            'migration',
            encode(b.bs_eui, 'hex'),
            format('Archived BSSCI operation %s (op %s)', t.command, t.op_id),
            format('Legacy operation tracking row %s archived by migration 000148 in state %s', t.id, t.state),
            s.basestation_id,
            jsonb_build_object(
                'legacyTable', 'bssci_operation_tracking',
                'legacyId', t.id,
                'sessionId', t.session_id,
                'opId', t.op_id,
                'command', t.command,
                'direction', t.direction,
                'state', t.state,
                'initiatedAt', t.initiated_at,
                'acknowledgedAt', t.acknowledged_at,
                'completedAt', t.completed_at,
                'responseData', t.response_data,
                'errorMessage', t.error_message,
                'createdAt', t.created_at
            ),
            t.initiated_at,
            NOW(),
            'resolved'
        FROM bssci_operation_tracking t
        JOIN basestation_sessions s ON s.id = t.session_id
        LEFT JOIN basestations b ON b.id = s.basestation_id
        RETURNING 1
    )
    SELECT count(*) INTO archived_count FROM archived;

    IF archived_count <> source_count THEN
        RAISE EXCEPTION 'bssci_operation_tracking archive mismatch: % source row(s), % archived', source_count, archived_count;
    END IF;
END $$;

DROP INDEX IF EXISTS idx_bssci_operation_pending;
DROP INDEX IF EXISTS idx_bssci_operation_session;

DROP TABLE IF EXISTS bssci_operation_tracking;
