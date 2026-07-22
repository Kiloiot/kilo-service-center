-- Migration 000140: purge orphaned BSSCI pending operations by session ownership.
--
-- Pending operations are reissued on session resume, so a row is only safe to
-- delete when its owning session can never resume: a terminated session, or one
-- explicitly marked non-resumable (can_resume = false). Rows of active and
-- disconnected-resumable sessions are preserved. Deletability is never inferred
-- from operation_type - the historical status-poll leak is reconciled at
-- startup against proven-terminal sessions, not here.
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
