-- Add missing endpoint fields for MIOTY compliance
ALTER TABLE endpoints 
ADD COLUMN IF NOT EXISTS manufacturer VARCHAR(255),
ADD COLUMN IF NOT EXISTS model VARCHAR(255),
ADD COLUMN IF NOT EXISTS carrier_offset INTEGER DEFAULT 0,
ADD COLUMN IF NOT EXISTS type_eui BYTEA;

-- Add check constraint for type_eui length (8 bytes = 64 bits)
ALTER TABLE endpoints 
ADD CONSTRAINT endpoints_type_eui_check CHECK (type_eui IS NULL OR length(type_eui) = 8);