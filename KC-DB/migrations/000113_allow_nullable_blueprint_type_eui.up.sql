-- Allow blueprints to have NULL type_eui when neither the request nor the
-- device model provides a TypeEUI value.
ALTER TABLE blueprints ALTER COLUMN type_eui DROP NOT NULL;
ALTER TABLE blueprints DROP CONSTRAINT IF EXISTS blueprints_type_eui_check;
ALTER TABLE blueprints ADD CONSTRAINT blueprints_type_eui_check
    CHECK (type_eui IS NULL OR length(type_eui) = 8);
