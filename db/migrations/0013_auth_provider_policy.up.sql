ALTER TABLE tenant_auth_methods
    ADD COLUMN config JSONB NOT NULL DEFAULT '{}'::jsonb;

ALTER TABLE tenants
    ADD COLUMN default_auth_method tenant_auth_method;
