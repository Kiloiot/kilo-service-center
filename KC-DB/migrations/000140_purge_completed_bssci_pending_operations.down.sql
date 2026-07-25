-- Migration 000140 down: documented no-op.
--
-- 000140 deletes pending-operation rows whose owning sessions can never resume.
-- That operational debris cannot be reconstructed, so the downgrade intentionally
-- does nothing (precedent: migration 101). The absence of the purged rows is
-- harmless - they would have been ignored on resume anyway.
SELECT 1;
