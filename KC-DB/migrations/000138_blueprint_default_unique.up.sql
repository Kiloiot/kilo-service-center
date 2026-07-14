-- Pre-dedupe or the UNIQUE index below fails on models that raced into 0/2 defaults.
UPDATE blueprints
   SET is_default = false, updated_at = NOW()
 WHERE is_default = true
   AND id NOT IN (
       SELECT DISTINCT ON (device_model_id) id
         FROM blueprints
        WHERE is_default = true
        ORDER BY device_model_id, created_at DESC
   );

DROP INDEX IF EXISTS idx_blueprints_default;
CREATE UNIQUE INDEX uq_blueprints_default_per_model ON blueprints(device_model_id) WHERE is_default;
