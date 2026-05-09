-- Restore default_auth_method as the enum type.
ALTER TABLE tenants ALTER COLUMN default_auth_method TYPE tenant_auth_method
    USING NULLIF(default_auth_method, '')::tenant_auth_method;

-- Drop the new tables and their enum.
DROP TABLE oidc_connections;
DROP TABLE social_connections;
DROP TYPE social_provider_kind;
