-- Migration 000140: purge orphaned BSSCI pending operations.
--
-- Two independent rules:
--
-- 1. Session ownership: pending operations are reissued on session resume, so
--    a row is safe to delete when its owning session can never resume - a
--    terminated session, or one explicitly marked non-resumable
--    (can_resume = false). Rows of active and disconnected-resumable sessions
--    are preserved regardless of type.
--
-- 2. Historical status-poll leak: before the completion fix (commit 31db76be,
--    2026-07-21 18:36:09+00) every status poll leaked its pending row because
--    the service center never finalized its own SC-initiated operations.
--    Status polls are idempotent liveness requests that are freshly issued on
--    every (re)connect, so pre-fix rows with operation_type = 'status' are
--    safely deleted even under resumable sessions. The boundary is the fixed
--    commit timestamp, never NOW(): the same rows are deleted no matter when
--    the migration runs.
--
-- Pre/post counts by session status and operation type are emitted as notices
-- for the migration evidence log. The down migration is a documented no-op:
-- deleted operational debris cannot be reconstructed.

DO $$
DECLARE
    rec RECORD;
    deletable BIGINT;
BEGIN
    RAISE NOTICE 'KC-MIG-000140: pending operations by session status BEFORE purge:';
    FOR rec IN
        SELECT s.status, po.operation_type, COUNT(*) AS n
        FROM bssci_pending_operations po
        JOIN basestation_sessions s ON s.id = po.basestation_session_id
        GROUP BY s.status, po.operation_type
        ORDER BY s.status, po.operation_type
    LOOP
        RAISE NOTICE '  status=% operation_type=% count=%', rec.status, rec.operation_type, rec.n;
    END LOOP;

    SELECT COUNT(*) INTO deletable
    FROM bssci_pending_operations po
    JOIN basestation_sessions s ON s.id = po.basestation_session_id
    WHERE s.status = 'terminated' OR s.can_resume = false;

    DELETE FROM bssci_pending_operations po
    USING basestation_sessions s
    WHERE po.basestation_session_id = s.id
      AND (s.status = 'terminated' OR s.can_resume = false);

    RAISE NOTICE 'KC-MIG-000140: purged % pending operation(s) of terminated/non-resumable sessions', deletable;

    -- Historical status-poll leak: pre-completion-fix status rows are
    -- idempotent polls reissued on every (re)connect
    SELECT COUNT(*) INTO deletable
    FROM bssci_pending_operations po
    WHERE po.operation_type = 'status'
      AND po.created_at < TIMESTAMPTZ '2026-07-21 18:36:09+00';

    DELETE FROM bssci_pending_operations po
    WHERE po.operation_type = 'status'
      AND po.created_at < TIMESTAMPTZ '2026-07-21 18:36:09+00';

    RAISE NOTICE 'KC-MIG-000140: purged % leaked pre-fix status poll row(s)', deletable;

    RAISE NOTICE 'KC-MIG-000140: pending operations by session status AFTER purge:';
    FOR rec IN
        SELECT s.status, po.operation_type, COUNT(*) AS n
        FROM bssci_pending_operations po
        JOIN basestation_sessions s ON s.id = po.basestation_session_id
        GROUP BY s.status, po.operation_type
        ORDER BY s.status, po.operation_type
    LOOP
        RAISE NOTICE '  status=% operation_type=% count=%', rec.status, rec.operation_type, rec.n;
    END LOOP;
END $$;
