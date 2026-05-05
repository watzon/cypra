-- Phase 6 personal access tokens.

CREATE TABLE personal_access_tokens (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    name TEXT NOT NULL,
    token_hash BYTEA NOT NULL UNIQUE,
    token_suffix TEXT NOT NULL,
    scopes TEXT[] NOT NULL DEFAULT ARRAY[]::TEXT[],
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    last_used_at TIMESTAMPTZ NULL,
    revoked_at TIMESTAMPTZ NULL
);

CREATE INDEX personal_access_tokens_tenant_id_idx ON personal_access_tokens (tenant_id);
CREATE INDEX personal_access_tokens_user_id_idx ON personal_access_tokens (user_id);

ALTER TABLE personal_access_tokens ENABLE ROW LEVEL SECURITY;
CREATE POLICY tenant_isolation ON personal_access_tokens USING (tenant_id = NULLIF(current_setting('cypra.tenant_id', true), '')::uuid);

GRANT ALL PRIVILEGES ON personal_access_tokens TO cypra_migrate;
GRANT SELECT, INSERT, UPDATE ON personal_access_tokens TO cypra_runtime;
