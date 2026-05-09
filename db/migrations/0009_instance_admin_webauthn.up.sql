-- Phase 17 instance-admin WebAuthn credentials for setup and dashboard auth.

CREATE TABLE instance_admin_passkey_credentials (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    instance_admin_id UUID NOT NULL REFERENCES instance_admins(id) ON DELETE CASCADE,
    credential_id BYTEA NOT NULL UNIQUE,
    public_key BYTEA NOT NULL,
    sign_count BIGINT NOT NULL DEFAULT 0,
    transports TEXT[] NOT NULL DEFAULT ARRAY[]::TEXT[],
    aaguid UUID NULL,
    rp_id TEXT NOT NULL,
    nickname TEXT NOT NULL DEFAULT '',
    credential JSONB NOT NULL DEFAULT '{}'::jsonb,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    last_used_at TIMESTAMPTZ NULL
);
CREATE INDEX instance_admin_passkey_credentials_admin_idx ON instance_admin_passkey_credentials (instance_admin_id);

CREATE TABLE instance_admin_webauthn_challenges (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    instance_admin_id UUID NOT NULL,
    kind TEXT NOT NULL CHECK (kind IN ('instance_admin_passkey_registration', 'instance_admin_passkey_assertion')),
    rp_id TEXT NOT NULL,
    session_data JSONB NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    expires_at TIMESTAMPTZ NOT NULL,
    consumed_at TIMESTAMPTZ NULL
);
CREATE INDEX instance_admin_webauthn_challenges_live_idx ON instance_admin_webauthn_challenges (kind, expires_at) WHERE consumed_at IS NULL;

GRANT ALL PRIVILEGES ON instance_admin_passkey_credentials TO cypra_migrate;
GRANT ALL PRIVILEGES ON instance_admin_webauthn_challenges TO cypra_migrate;
GRANT SELECT, INSERT, UPDATE ON instance_admin_passkey_credentials TO cypra_runtime;
GRANT SELECT, INSERT, UPDATE ON instance_admin_webauthn_challenges TO cypra_runtime;
