DROP INDEX IF EXISTS idx_federation_receipts_message_id;

ALTER TABLE federation_receipts
    DROP CONSTRAINT IF EXISTS fk_federation_receipts_message_id;
