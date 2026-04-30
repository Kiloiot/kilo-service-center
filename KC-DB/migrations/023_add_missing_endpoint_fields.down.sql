-- Remove added endpoint fields
ALTER TABLE endpoints 
DROP COLUMN IF EXISTS manufacturer,
DROP COLUMN IF EXISTS model,
DROP COLUMN IF EXISTS carrier_offset,
DROP COLUMN IF EXISTS type_eui;