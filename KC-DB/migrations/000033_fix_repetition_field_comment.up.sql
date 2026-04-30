-- Fix the incorrect comment on repetition field
-- The field stores the NUMBER of repetitions (1-15), not a boolean

COMMENT ON COLUMN endpoints.repetition IS 'Number of downlink repetitions (1-15) per MIOTY spec';