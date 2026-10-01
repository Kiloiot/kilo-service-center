-- A ref may name several downlinks again; the refs the up migration cleared
-- stay cleared.

DROP INDEX IF EXISTS uq_downlink_queue_command_ref;
