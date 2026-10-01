-- Retire the unused endpoint_keys subsystem.
--
-- Endpoint key material lives on the endpoints table as AES-256-GCM envelopes
-- (endpoints.nwk_key / endpoints.app_key, see migration 000143 and
-- pkg/keycrypto). The endpoint_keys table, its archive, and the standalone key
-- repository are dead: no production code reads or writes them.
--
-- Guard strategy: EMPTY-TABLES-ONLY. endpoint_keys.key_value holds raw
-- historical key bytes that pure SQL can neither decrypt-compare against the
-- endpoints envelopes nor migrate into them. Any remaining row - active or
-- inactive, of any key_type, in either table - therefore aborts the
-- migration: the KC-DB rekey command (KC-DB/cmd/rekey, apply mode)
-- reconciles live rows against the endpoints envelopes and exports the
-- archive to an encrypted backup, leaving both tables empty. Zero rows is the
-- proof that nothing can be lost by the drop.

BEGIN;

DO $$
DECLARE
    live_count bigint;
    archive_count bigint := 0;
BEGIN
    SELECT count(*) INTO live_count FROM endpoint_keys;
    IF to_regclass('endpoint_keys_archive') IS NOT NULL THEN
        SELECT count(*) INTO archive_count FROM endpoint_keys_archive;
    END IF;

    IF live_count > 0 OR archive_count > 0 THEN
        RAISE EXCEPTION
            'Cannot drop endpoint_keys: % live row(s) and % archive row(s) remain. Run the KC-DB rekey command (apply mode) to reconcile and export them, then re-run this migration.',
            live_count, archive_count;
    END IF;
END $$;

-- endpoint_keys_archive was created LIKE endpoint_keys; its only trigger is the
-- archive timestamp setter. set_archived_at() is shared with the other archive
-- tables and is intentionally left in place.
DROP TRIGGER IF EXISTS set_endpoint_keys_archive_timestamp ON endpoint_keys_archive;
DROP TABLE IF EXISTS endpoint_keys_archive;

-- endpoint_keys carries only a self-referential rotated_from foreign key; no
-- other table references it. Dropping the table removes its indexes.
DROP TABLE IF EXISTS endpoint_keys;

-- These functions were only ever bound to endpoint_keys triggers. Migration 015
-- already dropped those triggers when it rebuilt endpoint_keys, leaving the
-- functions orphaned; remove them now. update_audit_fields() and
-- set_archived_at() are shared and are kept.
DROP FUNCTION IF EXISTS validate_key_format();
DROP FUNCTION IF EXISTS ensure_single_active_key();

COMMIT;
