-- Rollback message partitioning
-- Only drop the utility functions created by the UP migration
-- The actual partitioning schema and tables are left intact (reversing partitioning is complex and error-prone)

-- Drop maintenance functions (in reverse order of creation)
DROP FUNCTION IF EXISTS drop_old_message_partitions(integer);
DROP FUNCTION IF EXISTS maintain_message_partitions(integer);
DROP FUNCTION IF EXISTS auto_create_monthly_partition();

-- Note: The UP migration doesn't create messages_auto_partition trigger or create_monthly_partition function,
-- so we don't try to drop them here.
-- The messages table partitioning remains in place (created in migration 001).
