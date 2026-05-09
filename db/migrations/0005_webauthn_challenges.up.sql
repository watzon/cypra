-- Phase 16 WebAuthn ceremonies.

ALTER TABLE passkey_credentials
    ADD COLUMN purpose TEXT NOT NULL DEFAULT 'primary' CHECK (purpose IN ('primary', 'second_factor')),
    ADD COLUMN credential JSONB NOT NULL DEFAULT '{}'::jsonb;

CREATE INDEX passkey_credentials_tenant_purpose_idx ON passkey_credentials (tenant_id, purpose);

CREATE TABLE webauthn_challenges (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    user_id UUID NULL REFERENCES users(id) ON DELETE CASCADE,
    kind TEXT NOT NULL CHECK (kind IN ('passkey_registration', 'passkey_assertion', 'webauthn2fa_registration', 'webauthn2fa_assertion')),
    rp_id TEXT NOT NULL,
    session_data JSONB NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    expires_at TIMESTAMPTZ NOT NULL,
    consumed_at TIMESTAMPTZ NULL
);

CREATE INDEX webauthn_challenges_tenant_id_idx ON webauthn_challenges (tenant_id);
CREATE INDEX webauthn_challenges_live_idx ON webauthn_challenges (tenant_id, kind, expires_at) WHERE consumed_at IS NULL;

ALTER TABLE webauthn_challenges ENABLE ROW LEVEL SECURITY;
CREATE POLICY tenant_isolation ON webauthn_challenges USING (tenant_id = NULLIF(current_setting('cypra.tenant_id', true), '')::uuid);

GRANT ALL PRIVILEGES ON webauthn_challenges TO cypra_migrate;
GRANT SELECT, INSERT, UPDATE ON webauthn_challenges TO cypra_runtime;
GRANT SELECT, INSERT, UPDATE ON passkey_credentials TO cypra_runtime;

CREATE TABLE invite_continuations (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NULL REFERENCES tenants(id) ON DELETE CASCADE,
    invite_id UUID NOT NULL REFERENCES pending_invitations(id) ON DELETE CASCADE,
    token_hash BYTEA NOT NULL,
    email CITEXT NOT NULL,
    role invite_role NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    expires_at TIMESTAMPTZ NOT NULL,
    consumed_at TIMESTAMPTZ NULL
);

CREATE INDEX invite_continuations_tenant_id_idx ON invite_continuations (tenant_id);
CREATE INDEX invite_continuations_live_idx ON invite_continuations (invite_id, expires_at) WHERE consumed_at IS NULL;

ALTER TABLE invite_continuations ENABLE ROW LEVEL SECURITY;
CREATE POLICY tenant_isolation ON invite_continuations USING (tenant_id IS NULL OR tenant_id = NULLIF(current_setting('cypra.tenant_id', true), '')::uuid);

GRANT ALL PRIVILEGES ON invite_continuations TO cypra_migrate;
GRANT SELECT, INSERT, UPDATE ON invite_continuations TO cypra_runtime;
