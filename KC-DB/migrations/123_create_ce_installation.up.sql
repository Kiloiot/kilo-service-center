-- CE installation singleton: stores identity and federation token for Community Edition instances.
-- The CONSTRAINT singleton enforces at most one row (id must equal 1).
CREATE TABLE IF NOT EXISTS ce_installation (
    id                      INTEGER PRIMARY KEY DEFAULT 1,
    ce_id                   UUID NOT NULL UNIQUE,
    company_name            VARCHAR(255) NOT NULL,
    onboarding_completed_at TIMESTAMPTZ,
    federation_token        TEXT,
    token_issued_at         TIMESTAMPTZ,
    created_at              TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at              TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT ce_installation_singleton CHECK (id = 1)
);
