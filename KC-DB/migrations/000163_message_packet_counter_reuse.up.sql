-- Record whether an uplink reused a packet counter the endpoint had already
-- sent outside the duplicate window (SCACI §3.8.1 duplicate).
--
-- messages.duplicate stays the published "received by several base stations"
-- flag the API serves; a reused counter is a different fact, so it gets its own
-- column. The archive mirrors the live layout (000139).

ALTER TABLE messages ADD COLUMN packet_cnt_reused BOOLEAN NOT NULL DEFAULT false;
ALTER TABLE messages_archive ADD COLUMN packet_cnt_reused BOOLEAN NOT NULL DEFAULT false;

COMMENT ON COLUMN messages.packet_cnt_reused IS
    'True when the packet counter was already used by an earlier telegram outside the duplicate window (SCACI §3.8.1 duplicate)';
COMMENT ON COLUMN messages.duplicate IS
    'True once a further base station reception of the telegram merged into the message';
