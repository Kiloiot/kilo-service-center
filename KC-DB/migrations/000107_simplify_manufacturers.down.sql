-- Migration 000107 down: Restore manufacturers columns (conservative rollback)
-- Columns restored as NULLABLE to avoid data issues on rollback

-- Restore columns as NULLABLE
ALTER TABLE manufacturers ADD COLUMN code VARCHAR(64);
ALTER TABLE manufacturers ADD COLUMN description TEXT;
ALTER TABLE manufacturers ADD COLUMN logo_url VARCHAR(512);

-- Remove new constraint
DROP INDEX IF EXISTS uq_manufacturers_tenant_name_ci;

-- Restore code index (no uniqueness to avoid slugify collision issues on rollback)
CREATE INDEX idx_manufacturers_code ON manufacturers(code);

-- Note: code remains NULLABLE after rollback. Original constraint (NOT NULL + UNIQUE)
-- is NOT restored because simple slugify can produce collisions (e.g., "ACME!" and "ACME?")

-- Restore original comment
COMMENT ON TABLE manufacturers IS 'Device manufacturers for blueprint catalog';
COMMENT ON COLUMN manufacturers.code IS 'URL-friendly slug identifier (nullable after rollback)';
