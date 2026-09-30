-- Drop the orphaned mioty_subpackets table (018).
--
-- Its parent, mioty_messages, was dropped with CASCADE in 000047, which
-- removed the composite foreign key and left this child table with no
-- writer: subpacket metrics live in messages.subpackets JSONB since then.
-- The table must be empty; a populated table means an unknown writer exists
-- and the migration aborts for review.

DO $$
DECLARE
    row_count bigint;
BEGIN
    SELECT count(*) INTO row_count FROM mioty_subpackets;
    IF row_count > 0 THEN
        RAISE EXCEPTION 'mioty_subpackets holds % row(s) although its parent table was dropped; investigate the writer before re-running', row_count;
    END IF;
END $$;

DROP INDEX IF EXISTS idx_mioty_subpackets_tenant;
DROP INDEX IF EXISTS idx_mioty_subpackets_message;

DROP TABLE IF EXISTS mioty_subpackets;
