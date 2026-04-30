-- Rollback: restore NOT NULL on type_eui.
-- Rows with NULL type_eui must be removed first.
DELETE FROM blueprints WHERE type_eui IS NULL;
ALTER TABLE blueprints DROP CONSTRAINT IF EXISTS blueprints_type_eui_check;
ALTER TABLE blueprints ADD CONSTRAINT blueprints_type_eui_check
    CHECK (length(type_eui) = 8);
ALTER TABLE blueprints ALTER COLUMN type_eui SET NOT NULL;
