-- Cypra Phase 1 initial schema.
-- The earlier PLAN data-model name `admin_invites` is implemented here as
-- `pending_invitations`, a unified table for tenant-admin, member, and
-- instance-admin invites.

CREATE EXTENSION IF NOT EXISTS citext;
CREATE EXTENSION IF NOT EXISTS pgcrypto;

CREATE TYPE membership_role AS ENUM ('owner', 'admin', 'member');
CREATE TYPE invite_role AS ENUM ('owner', 'admin', 'member', 'instance_admin');
CREATE TYPE invitation_creator_kind AS ENUM ('user', 'tenant_admin', 'instance_admin', 'system');
CREATE TYPE session_subject_kind AS ENUM ('user', 'tenant_admin');
CREATE TYPE oidc_auth_method AS ENUM ('client_secret_basic', 'client_secret_post', 'none');
CREATE TYPE pkce_method AS ENUM ('S256');
CREATE TYPE signing_key_state AS ENUM ('active', 'overlap', 'retired', 'sunsetting');
CREATE TYPE upstream_provider_kind AS ENUM ('google');
CREATE TYPE email_provider_kind AS ENUM ('terminal', 'smtp', 'resend');
CREATE TYPE storage_backend AS ENUM ('local-disk', 's3-compatible');
CREATE TYPE tenant_auth_method AS ENUM ('password', 'magic_link', 'passkey', 'totp', 'google', 'oidc_upstream');
CREATE TYPE audit_actor_kind AS ENUM ('user', 'tenant_admin', 'instance_admin', 'system');
CREATE TYPE master_key_rotation_phase AS ENUM ('rewrap', 'cutover', 'done');
CREATE TYPE gdpr_deletion_state AS ENUM ('pending', 'processing', 'done', 'failed');

CREATE TABLE tenants (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    slug TEXT NOT NULL,
    name TEXT NOT NULL,
    branding JSONB NOT NULL DEFAULT '{}'::jsonb,
    settings JSONB NOT NULL DEFAULT '{}'::jsonb,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    deleted_at TIMESTAMPTZ NULL,
    CONSTRAINT tenants_slug_format CHECK (slug ~ '^[a-z][a-z0-9-]{2,63}$')
);
CREATE UNIQUE INDEX tenants_slug_live_idx ON tenants (slug) WHERE deleted_at IS NULL;

CREATE TABLE projects (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    slug TEXT NOT NULL,
    name TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    deleted_at TIMESTAMPTZ NULL
);
CREATE UNIQUE INDEX projects_tenant_slug_live_idx ON projects (tenant_id, slug) WHERE deleted_at IS NULL;
CREATE INDEX projects_tenant_id_idx ON projects (tenant_id);

CREATE TABLE instance_admins (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    email CITEXT NOT NULL UNIQUE,
    display_name TEXT NOT NULL,
    metadata JSONB NOT NULL DEFAULT '{}'::jsonb,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    last_login_at TIMESTAMPTZ NULL,
    disabled_at TIMESTAMPTZ NULL
);

CREATE TABLE users (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    email CITEXT NOT NULL,
    email_verified_at TIMESTAMPTZ NULL,
    metadata JSONB NOT NULL DEFAULT '{}'::jsonb,
    profile_picture_object_id UUID NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    deleted_at TIMESTAMPTZ NULL
);
CREATE UNIQUE INDEX users_tenant_email_live_idx ON users (tenant_id, email) WHERE deleted_at IS NULL;
CREATE INDEX users_tenant_id_idx ON users (tenant_id);

CREATE TABLE tenant_memberships (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    role membership_role NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (tenant_id, user_id)
);
CREATE INDEX tenant_memberships_tenant_id_idx ON tenant_memberships (tenant_id);

CREATE TABLE password_credentials (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    argon2id_hash BYTEA NOT NULL,
    must_reset BOOLEAN NOT NULL DEFAULT false,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (user_id)
);
CREATE INDEX password_credentials_tenant_id_idx ON password_credentials (tenant_id);

CREATE TABLE passkey_credentials (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    credential_id BYTEA NOT NULL UNIQUE,
    public_key BYTEA NOT NULL,
    sign_count BIGINT NOT NULL DEFAULT 0,
    transports TEXT[] NOT NULL DEFAULT ARRAY[]::TEXT[],
    aaguid UUID NULL,
    rp_id TEXT NOT NULL,
    nickname TEXT NOT NULL DEFAULT '',
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    last_used_at TIMESTAMPTZ NULL
);
CREATE INDEX passkey_credentials_tenant_id_idx ON passkey_credentials (tenant_id);

CREATE TABLE totp_credentials (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    secret_encrypted BYTEA NOT NULL,
    algorithm TEXT NOT NULL DEFAULT 'SHA1',
    digits INT NOT NULL DEFAULT 6,
    period_seconds INT NOT NULL DEFAULT 30,
    confirmed_at TIMESTAMPTZ NULL
);
CREATE INDEX totp_credentials_tenant_id_idx ON totp_credentials (tenant_id);

CREATE TABLE user_backup_codes (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    code_hash BYTEA NOT NULL,
    used_at TIMESTAMPTZ NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX user_backup_codes_tenant_id_idx ON user_backup_codes (tenant_id);

CREATE TABLE instance_admin_backup_codes (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    instance_admin_id UUID NOT NULL REFERENCES instance_admins(id) ON DELETE CASCADE,
    code_hash BYTEA NOT NULL,
    used_at TIMESTAMPTZ NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE magic_link_tokens (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    user_id UUID NULL REFERENCES users(id) ON DELETE CASCADE,
    email CITEXT NOT NULL,
    token_hash BYTEA NOT NULL UNIQUE,
    expires_at TIMESTAMPTZ NOT NULL,
    consumed_at TIMESTAMPTZ NULL,
    revoked_at TIMESTAMPTZ NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX magic_link_tokens_tenant_user_expires_idx ON magic_link_tokens (tenant_id, user_id, expires_at);

CREATE TABLE password_reset_tokens (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    token_hash BYTEA NOT NULL UNIQUE,
    expires_at TIMESTAMPTZ NOT NULL,
    consumed_at TIMESTAMPTZ NULL,
    revoked_at TIMESTAMPTZ NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX password_reset_tokens_tenant_user_expires_idx ON password_reset_tokens (tenant_id, user_id, expires_at);

CREATE TABLE email_verification_tokens (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    token_hash BYTEA NOT NULL UNIQUE,
    expires_at TIMESTAMPTZ NOT NULL,
    consumed_at TIMESTAMPTZ NULL,
    revoked_at TIMESTAMPTZ NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX email_verification_tokens_tenant_user_expires_idx ON email_verification_tokens (tenant_id, user_id, expires_at);

CREATE TABLE pending_invitations (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NULL REFERENCES tenants(id) ON DELETE CASCADE,
    email CITEXT NOT NULL,
    role invite_role NOT NULL,
    token_hash BYTEA NOT NULL UNIQUE,
    created_by_kind invitation_creator_kind NOT NULL,
    created_by_id UUID NULL,
    expires_at TIMESTAMPTZ NOT NULL,
    redeemed_at TIMESTAMPTZ NULL,
    redeemed_by_user_id UUID NULL REFERENCES users(id) ON DELETE SET NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE UNIQUE INDEX pending_invitations_tenant_email_unredeemed_idx ON pending_invitations (tenant_id, email) WHERE redeemed_at IS NULL;
CREATE UNIQUE INDEX pending_invitations_instance_email_unredeemed_idx ON pending_invitations (email) WHERE tenant_id IS NULL AND redeemed_at IS NULL;

CREATE TABLE sessions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    subject_id UUID NOT NULL,
    subject_kind session_subject_kind NOT NULL,
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    last_seen_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    expires_at TIMESTAMPTZ NOT NULL,
    revoked_at TIMESTAMPTZ NULL,
    ip INET NULL,
    user_agent TEXT NULL
);
CREATE INDEX sessions_tenant_id_idx ON sessions (tenant_id);

CREATE TABLE instance_admin_sessions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    instance_admin_id UUID NOT NULL REFERENCES instance_admins(id) ON DELETE CASCADE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    last_seen_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    expires_at TIMESTAMPTZ NOT NULL,
    revoked_at TIMESTAMPTZ NULL,
    ip INET NULL,
    user_agent TEXT NULL
);

CREATE TABLE oidc_clients (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    project_id UUID NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
    client_id TEXT NOT NULL UNIQUE,
    client_secret_encrypted BYTEA NULL,
    redirect_uris TEXT[] NOT NULL,
    allowed_scopes TEXT[] NOT NULL,
    token_endpoint_auth_method oidc_auth_method NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    deleted_at TIMESTAMPTZ NULL
);
CREATE INDEX oidc_clients_tenant_id_idx ON oidc_clients (tenant_id);

CREATE TABLE oidc_authorization_codes (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    code_hash BYTEA NOT NULL UNIQUE,
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    oidc_client_uuid UUID NOT NULL REFERENCES oidc_clients(id) ON DELETE CASCADE,
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    redirect_uri TEXT NOT NULL,
    scope TEXT[] NOT NULL,
    pkce_challenge TEXT NOT NULL,
    pkce_method pkce_method NOT NULL,
    nonce TEXT NULL,
    expires_at TIMESTAMPTZ NOT NULL,
    consumed_at TIMESTAMPTZ NULL
);
CREATE INDEX oidc_authorization_codes_tenant_id_idx ON oidc_authorization_codes (tenant_id);

CREATE TABLE oidc_refresh_tokens (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    family_id UUID NOT NULL,
    parent_id UUID NULL REFERENCES oidc_refresh_tokens(id) ON DELETE SET NULL,
    token_hash BYTEA NOT NULL UNIQUE,
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    oidc_client_uuid UUID NOT NULL REFERENCES oidc_clients(id) ON DELETE CASCADE,
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    scope TEXT[] NOT NULL,
    expires_at TIMESTAMPTZ NOT NULL,
    consumed_at TIMESTAMPTZ NULL,
    revoked_at TIMESTAMPTZ NULL,
    revoke_reason TEXT NULL
);
CREATE INDEX oidc_refresh_tokens_tenant_id_idx ON oidc_refresh_tokens (tenant_id);
CREATE INDEX oidc_refresh_tokens_family_id_idx ON oidc_refresh_tokens (family_id);
CREATE INDEX oidc_refresh_tokens_token_hash_idx ON oidc_refresh_tokens (token_hash);

CREATE TABLE oidc_consents (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    oidc_client_uuid UUID NOT NULL REFERENCES oidc_clients(id) ON DELETE CASCADE,
    scopes TEXT[] NOT NULL,
    granted_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    revoked_at TIMESTAMPTZ NULL,
    UNIQUE (user_id, oidc_client_uuid)
);
CREATE INDEX oidc_consents_tenant_id_idx ON oidc_consents (tenant_id);

CREATE TABLE oidc_signing_keys (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE NO ACTION,
    kid TEXT NOT NULL UNIQUE,
    algorithm TEXT NOT NULL DEFAULT 'RS256',
    public_key_jwk JSONB NOT NULL,
    private_key_encrypted BYTEA NOT NULL,
    state signing_key_state NOT NULL,
    activated_at TIMESTAMPTZ NOT NULL,
    retires_at TIMESTAMPTZ NOT NULL,
    sunset_until TIMESTAMPTZ NULL
);
CREATE INDEX oidc_signing_keys_tenant_id_idx ON oidc_signing_keys (tenant_id);
CREATE UNIQUE INDEX oidc_signing_keys_one_active_idx ON oidc_signing_keys (tenant_id) WHERE state = 'active';
CREATE UNIQUE INDEX oidc_signing_keys_one_overlap_idx ON oidc_signing_keys (tenant_id) WHERE state = 'overlap';

CREATE TABLE upstream_providers (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    kind upstream_provider_kind NOT NULL,
    client_id_encrypted BYTEA NOT NULL,
    client_secret_encrypted BYTEA NOT NULL,
    enabled BOOLEAN NOT NULL DEFAULT false,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX upstream_providers_tenant_id_idx ON upstream_providers (tenant_id);

CREATE TABLE email_provider_configs (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    kind email_provider_kind NOT NULL,
    config_encrypted BYTEA NOT NULL,
    from_address TEXT NOT NULL,
    from_name TEXT NOT NULL
);
CREATE INDEX email_provider_configs_tenant_id_idx ON email_provider_configs (tenant_id);

CREATE TABLE email_outbox (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NULL REFERENCES tenants(id) ON DELETE CASCADE,
    to_address CITEXT NOT NULL,
    template TEXT NOT NULL,
    payload JSONB NOT NULL DEFAULT '{}'::jsonb,
    attempts INT NOT NULL DEFAULT 0,
    next_attempt_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    sent_at TIMESTAMPTZ NULL,
    failed_at TIMESTAMPTZ NULL,
    last_error TEXT NULL
);
CREATE INDEX email_outbox_next_attempt_idx ON email_outbox (next_attempt_at) WHERE sent_at IS NULL AND failed_at IS NULL;
CREATE INDEX email_outbox_tenant_id_idx ON email_outbox (tenant_id);

CREATE TABLE storage_objects (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    backend storage_backend NOT NULL,
    key TEXT NOT NULL,
    content_type TEXT NOT NULL,
    byte_size BIGINT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    deleted_at TIMESTAMPTZ NULL
);
CREATE INDEX storage_objects_tenant_id_idx ON storage_objects (tenant_id);

ALTER TABLE users
    ADD CONSTRAINT users_profile_picture_object_id_fkey
    FOREIGN KEY (profile_picture_object_id) REFERENCES storage_objects(id) ON DELETE SET NULL;

CREATE TABLE tenant_auth_methods (
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    method tenant_auth_method NOT NULL,
    enabled BOOLEAN NOT NULL DEFAULT false,
    enrolled_count_cache BIGINT NOT NULL DEFAULT 0,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    PRIMARY KEY (tenant_id, method)
);

CREATE TABLE audit_entries (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    occurred_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    tenant_id UUID NULL REFERENCES tenants(id) ON DELETE SET NULL,
    actor_kind audit_actor_kind NOT NULL,
    actor_id UUID NULL,
    action TEXT NOT NULL,
    resource_kind TEXT NOT NULL,
    resource_id UUID NULL,
    state_before JSONB NULL,
    state_after JSONB NULL,
    ip INET NULL,
    user_agent TEXT NULL,
    metadata JSONB NOT NULL DEFAULT '{}'::jsonb,
    redacted_at TIMESTAMPTZ NULL,
    CONSTRAINT audit_entries_redaction_shape CHECK (
        redacted_at IS NULL OR (state_before IS NOT NULL OR state_after IS NOT NULL OR metadata IS NOT NULL)
    )
);
CREATE INDEX audit_entries_tenant_occurred_idx ON audit_entries (tenant_id, occurred_at DESC);
CREATE INDEX audit_entries_action_occurred_idx ON audit_entries (action, occurred_at DESC);

CREATE TABLE bootstrap_tokens (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    token_hash BYTEA NOT NULL UNIQUE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    expires_at TIMESTAMPTZ NOT NULL,
    consumed_at TIMESTAMPTZ NULL,
    revoked_at TIMESTAMPTZ NULL
);
CREATE UNIQUE INDEX bootstrap_tokens_one_live_idx ON bootstrap_tokens ((true)) WHERE consumed_at IS NULL AND revoked_at IS NULL;

CREATE TABLE master_key_rotations (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    started_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    completed_at TIMESTAMPTZ NULL,
    phase master_key_rotation_phase NOT NULL,
    rows_total BIGINT NOT NULL DEFAULT 0,
    rows_done BIGINT NOT NULL DEFAULT 0,
    error TEXT NULL
);

CREATE TABLE rate_limit_buckets (
    scope TEXT NOT NULL,
    key TEXT NOT NULL,
    tenant_id UUID NULL REFERENCES tenants(id) ON DELETE CASCADE,
    tokens DOUBLE PRECISION NOT NULL,
    last_refill_at TIMESTAMPTZ NOT NULL
);
CREATE UNIQUE INDEX rate_limit_buckets_scope_key_tenant_idx ON rate_limit_buckets (scope, key, tenant_id) NULLS NOT DISTINCT;

CREATE TABLE gdpr_deletions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    state gdpr_deletion_state NOT NULL DEFAULT 'pending',
    requested_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    completed_at TIMESTAMPTZ NULL,
    error TEXT NULL
);
CREATE INDEX gdpr_deletions_tenant_id_idx ON gdpr_deletions (tenant_id);
