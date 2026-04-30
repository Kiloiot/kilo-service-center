-- Add refresh_tokens table for local auth
-- Supports JWT refresh token rotation with reuse detection

-- Ensure pgcrypto extension for gen_random_uuid()
CREATE EXTENSION IF NOT EXISTS pgcrypto;

-- Create refresh_tokens table
CREATE TABLE IF NOT EXISTS refresh_tokens (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    token_hash VARCHAR(64) NOT NULL UNIQUE,  -- SHA256 hex, pre-hashed by caller
    issued_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    expires_at TIMESTAMPTZ NOT NULL,
    revoked_at TIMESTAMPTZ,
    replaced_by UUID REFERENCES refresh_tokens(id),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Index for user lookup (family revocation)
CREATE INDEX IF NOT EXISTS idx_refresh_tokens_user_id ON refresh_tokens(user_id);

-- Index for expired token cleanup (only non-revoked tokens)
CREATE INDEX IF NOT EXISTS idx_refresh_tokens_expires ON refresh_tokens(expires_at) WHERE revoked_at IS NULL;

COMMENT ON TABLE refresh_tokens IS 'JWT refresh tokens with rotation and reuse detection';
COMMENT ON COLUMN refresh_tokens.token_hash IS 'SHA256 hex hash of the refresh token (never store raw tokens)';
COMMENT ON COLUMN refresh_tokens.replaced_by IS 'Points to the new token that replaced this one during rotation';
COMMENT ON COLUMN refresh_tokens.revoked_at IS 'Set when token is explicitly revoked or family is invalidated';
