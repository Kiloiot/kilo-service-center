-- A session holds one ulData operation per stored uplink.
--
-- The delivery worker retries an uplink's SCACI row as a whole when any
-- Application Center session did not get it, and each attempt recorded a new
-- ulData operation under a new opId for every session, including those that
-- had received or recorded it before: an Application Center got the telegram
-- again as a new operation, with duplicate=false and the same packet counter
-- (SCACI §3.2 keeps an operation's opId; §3.8.1 marks a reused counter
-- duplicate; §1 reissues only what was not completed). An outbound ulData
-- record now carries the stored uplink it delivers under
-- request_data.sourceMessageId (never sent on the wire), and this index keeps
-- one such record per session and uplink, so a repeated attempt finds the
-- session's original operation instead of recording another.
--
-- The records stored before carry no sourceMessageId and stay outside the
-- index: no existing row can violate it.
--
-- Locking: CREATE INDEX takes a SHARE lock on scaci_operation_log while it
-- scans the table, holding back its writes (not its reads). The preflight row
-- "000186 ..." counts the rows the build scans and the open writers.
--
-- Idempotent: the index is created only when missing.

CREATE UNIQUE INDEX IF NOT EXISTS uq_scaci_op_log_uplink_delivery
    ON scaci_operation_log (session_id, (request_data->>'sourceMessageId'))
    WHERE command = 'ulData' AND direction = 'outbound' AND (request_data->>'sourceMessageId') IS NOT NULL;
