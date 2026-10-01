-- Drop the downlink window claim. Earlier releases dispatch per reception and
-- never read the column; windows already used stay used.

ALTER TABLE messages_archive DROP COLUMN IF EXISTS dl_window_claimed;
ALTER TABLE messages DROP COLUMN IF EXISTS dl_window_claimed;
