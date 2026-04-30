-- Revert case-insensitive email normalization
DROP INDEX IF EXISTS identity.users_email_lower_unique;
ALTER TABLE identity.users ADD CONSTRAINT users_email_key UNIQUE (email);
