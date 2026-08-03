-- Email verification: users.verified_at is null until the owner proves
-- control of their email by presenting a token issued through
-- email_verification_tokens (same shape/spirit as refresh_tokens — the raw
-- token is never stored, only its SHA-256 hash).

ALTER TABLE users ADD COLUMN verified_at TIMESTAMPTZ;

CREATE TABLE email_verification_tokens (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    token_hash TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    expires_at TIMESTAMPTZ NOT NULL,
    used_at TIMESTAMPTZ
);

CREATE INDEX idx_email_verification_tokens_user_id ON email_verification_tokens (user_id);
CREATE UNIQUE INDEX idx_email_verification_tokens_token_hash ON email_verification_tokens (token_hash);
