-- Remove the placeholder statistics table and its refresh function.
--
-- table_statistics (010) only ever received rows from update_table_statistics(),
-- whose body counts "SELECT COUNT(*) FROM public.messages WHERE false" for
-- every table, so the column values were always zero. No repository, service
-- or scheduled job calls the function, and PostgreSQL's own pg_stat_* views
-- carry the real numbers. The data is derived, so there is nothing to
-- preserve and no guard is needed.

DROP FUNCTION IF EXISTS update_table_statistics();

DROP TABLE IF EXISTS table_statistics;
