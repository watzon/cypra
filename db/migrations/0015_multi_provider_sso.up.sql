-- Foundation for multi-provider SSO (Core 5: Google, Microsoft, Apple, GitHub, Discord)
-- and multi-instance Enterprise OIDC. This migration is additive: existing
-- upstream_providers rows + tenant_auth_methods enum stay intact and continue
-- to drive the live Google flow. Subsequent PRs will move the runtime over to
-- social_connections / oidc_connections and then drop the legacy table + enum
-- values.

-- 1. New social_provider_kind enum (covers Core 5).
CREATE TYPE social_provider_kind AS ENUM ('google', 'microsoft', 'apple', 'github', 'discord');

-- 2. Replacement table for upstream_providers. Carries per-connection config
--    JSONB for allowed_domains / allow_signup / future per-provider knobs.
CREATE TABLE social_connections (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    kind social_provider_kind NOT NULL,
    enabled BOOLEAN NOT NULL DEFAULT false,
    client_id_encrypted BYTEA NOT NULL,
    client_secret_encrypted BYTEA NOT NULL,
    config JSONB NOT NULL DEFAULT '{}'::jsonb,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (tenant_id, kind)
);
ALTER TABLE social_connections ENABLE ROW LEVEL SECURITY;
CREATE POLICY tenant_isolation ON social_connections USING (tenant_id = NULLIF(current_setting('cypra.tenant_id', true), '')::uuid);

-- 3. Mirror existing Google rows into social_connections so the new code path
--    can read from either table during transition. Uses LEFT JOIN to also
--    carry the previously-stored allow_signup / allowed_domains policy that
--    lived on tenant_auth_methods.config keyed to method='google'.
INSERT INTO social_connections (tenant_id, kind, enabled, client_id_encrypted, client_secret_encrypted, config)
SELECT
    up.tenant_id,
    'google'::social_provider_kind,
    up.enabled,
    up.client_id_encrypted,
    up.client_secret_encrypted,
    COALESCE(tam.config, '{}'::jsonb)
FROM upstream_providers up
LEFT JOIN tenant_auth_methods tam
    ON tam.tenant_id = up.tenant_id AND tam.method::text = 'google';

-- 4. Multi-instance enterprise OIDC. Slug-keyed so a tenant can run several
--    upstream OIDC connections (Okta-prod, Auth0-staging, etc.) side by side.
CREATE TABLE oidc_connections (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    slug TEXT NOT NULL,
    display_name TEXT NOT NULL,
    issuer_url TEXT NOT NULL,
    client_id_encrypted BYTEA NOT NULL,
    client_secret_encrypted BYTEA NOT NULL,
    scopes TEXT[] NOT NULL DEFAULT ARRAY['openid','email','profile'],
    config JSONB NOT NULL DEFAULT '{}'::jsonb,
    enabled BOOLEAN NOT NULL DEFAULT false,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (tenant_id, slug)
);
ALTER TABLE oidc_connections ENABLE ROW LEVEL SECURITY;
CREATE POLICY tenant_isolation ON oidc_connections USING (tenant_id = NULLIF(current_setting('cypra.tenant_id', true), '')::uuid);

-- 5. tenants.default_auth_method becomes TEXT so it can carry "social:google",
--    "oidc:<slug>", or a built-in identifier. The enum values still exist as
--    string literals; the server enforces the wider format on write.
ALTER TABLE tenants ALTER COLUMN default_auth_method DROP DEFAULT;
ALTER TABLE tenants ALTER COLUMN default_auth_method TYPE TEXT USING default_auth_method::text;
