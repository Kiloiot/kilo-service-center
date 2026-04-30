ALTER TABLE federation_receipts
    ADD CONSTRAINT fk_federation_receipts_message_id
    FOREIGN KEY (message_id) REFERENCES messages(id);

CREATE INDEX idx_federation_receipts_message_id
    ON federation_receipts(message_id);
