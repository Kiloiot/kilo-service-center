-- Remove redundant ep_eui field from endpoints table
-- In MIOTY, each endpoint has exactly one EUI (stored in the eui column)
-- The ep_eui field was added by mistake and is not needed

ALTER TABLE endpoints DROP COLUMN IF EXISTS ep_eui;

-- Also remove any references from the endpoint model if they exist
COMMENT ON COLUMN endpoints.eui IS 'Endpoint EUI (8-byte Extended Unique Identifier) - serves as the endpoint identifier in MIOTY';
