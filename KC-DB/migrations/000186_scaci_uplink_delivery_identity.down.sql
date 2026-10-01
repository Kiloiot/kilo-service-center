-- The previous release records a new ulData operation on every delivery
-- attempt; the records keep their sourceMessageId, which it ignores.
DROP INDEX IF EXISTS uq_scaci_op_log_uplink_delivery;
