-- A BSSCI base station registered while the server knows no URL a station can
-- reach (a wildcard or loopback BSSCI host with no external URL, the default
-- local install) stores its Service Center URL as NULL, and the UI shows that
-- state with a warning. check_bssci_config refused those rows, so registering
-- such a station failed.
--
-- Locking: dropping a CHECK constraint is a catalog change under a brief
-- ACCESS EXCLUSIVE lock; no row is read or rewritten.

ALTER TABLE basestations DROP CONSTRAINT IF EXISTS check_bssci_config;
