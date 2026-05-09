ALTER TABLE tenants
    DROP COLUMN invites_enabled,
    DROP COLUMN signup_allowlist,
    DROP COLUMN signup_mode;

DROP TYPE tenant_signup_mode;
