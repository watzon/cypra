ALTER TABLE tenants
    DROP COLUMN default_auth_method;

ALTER TABLE tenant_auth_methods
    DROP COLUMN config;
