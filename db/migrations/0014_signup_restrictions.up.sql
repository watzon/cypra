CREATE TYPE tenant_signup_mode AS ENUM ('open', 'restricted', 'closed');

ALTER TABLE tenants
    ADD COLUMN signup_mode tenant_signup_mode NOT NULL DEFAULT 'open',
    ADD COLUMN signup_allowlist TEXT[] NOT NULL DEFAULT '{}',
    ADD COLUMN invites_enabled BOOLEAN NOT NULL DEFAULT true;
