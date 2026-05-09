ALTER TABLE personal_access_tokens
    ADD COLUMN expires_at TIMESTAMPTZ NULL;

CREATE INDEX personal_access_tokens_live_idx ON personal_access_tokens (token_hash)
    WHERE revoked_at IS NULL;
