-- Row-level security is the database belt behind TenantScopedDB. Runtime
-- connections see only rows whose tenant_id matches cypra.tenant_id; migrate
-- role bypasses RLS for schema and data migrations.

ALTER ROLE cypra_migrate BYPASSRLS;

ALTER TABLE projects ENABLE ROW LEVEL SECURITY;
CREATE POLICY tenant_isolation ON projects USING (tenant_id = NULLIF(current_setting('cypra.tenant_id', true), '')::uuid);

ALTER TABLE tenant_memberships ENABLE ROW LEVEL SECURITY;
CREATE POLICY tenant_isolation ON tenant_memberships USING (tenant_id = NULLIF(current_setting('cypra.tenant_id', true), '')::uuid);

ALTER TABLE users ENABLE ROW LEVEL SECURITY;
CREATE POLICY tenant_isolation ON users USING (tenant_id = NULLIF(current_setting('cypra.tenant_id', true), '')::uuid);

ALTER TABLE password_credentials ENABLE ROW LEVEL SECURITY;
CREATE POLICY tenant_isolation ON password_credentials USING (tenant_id = NULLIF(current_setting('cypra.tenant_id', true), '')::uuid);

ALTER TABLE passkey_credentials ENABLE ROW LEVEL SECURITY;
CREATE POLICY tenant_isolation ON passkey_credentials USING (tenant_id = NULLIF(current_setting('cypra.tenant_id', true), '')::uuid);

ALTER TABLE totp_credentials ENABLE ROW LEVEL SECURITY;
CREATE POLICY tenant_isolation ON totp_credentials USING (tenant_id = NULLIF(current_setting('cypra.tenant_id', true), '')::uuid);

ALTER TABLE user_backup_codes ENABLE ROW LEVEL SECURITY;
CREATE POLICY tenant_isolation ON user_backup_codes USING (tenant_id = NULLIF(current_setting('cypra.tenant_id', true), '')::uuid);

ALTER TABLE magic_link_tokens ENABLE ROW LEVEL SECURITY;
CREATE POLICY tenant_isolation ON magic_link_tokens USING (tenant_id = NULLIF(current_setting('cypra.tenant_id', true), '')::uuid);

ALTER TABLE password_reset_tokens ENABLE ROW LEVEL SECURITY;
CREATE POLICY tenant_isolation ON password_reset_tokens USING (tenant_id = NULLIF(current_setting('cypra.tenant_id', true), '')::uuid);

ALTER TABLE email_verification_tokens ENABLE ROW LEVEL SECURITY;
CREATE POLICY tenant_isolation ON email_verification_tokens USING (tenant_id = NULLIF(current_setting('cypra.tenant_id', true), '')::uuid);

ALTER TABLE pending_invitations ENABLE ROW LEVEL SECURITY;
CREATE POLICY tenant_isolation ON pending_invitations USING (tenant_id IS NULL OR tenant_id = NULLIF(current_setting('cypra.tenant_id', true), '')::uuid);

ALTER TABLE sessions ENABLE ROW LEVEL SECURITY;
CREATE POLICY tenant_isolation ON sessions USING (tenant_id = NULLIF(current_setting('cypra.tenant_id', true), '')::uuid);

ALTER TABLE oidc_clients ENABLE ROW LEVEL SECURITY;
CREATE POLICY tenant_isolation ON oidc_clients USING (tenant_id = NULLIF(current_setting('cypra.tenant_id', true), '')::uuid);

ALTER TABLE oidc_authorization_codes ENABLE ROW LEVEL SECURITY;
CREATE POLICY tenant_isolation ON oidc_authorization_codes USING (tenant_id = NULLIF(current_setting('cypra.tenant_id', true), '')::uuid);

ALTER TABLE oidc_refresh_tokens ENABLE ROW LEVEL SECURITY;
CREATE POLICY tenant_isolation ON oidc_refresh_tokens USING (tenant_id = NULLIF(current_setting('cypra.tenant_id', true), '')::uuid);

ALTER TABLE oidc_consents ENABLE ROW LEVEL SECURITY;
CREATE POLICY tenant_isolation ON oidc_consents USING (tenant_id = NULLIF(current_setting('cypra.tenant_id', true), '')::uuid);

ALTER TABLE oidc_signing_keys ENABLE ROW LEVEL SECURITY;
CREATE POLICY tenant_isolation ON oidc_signing_keys USING (tenant_id = NULLIF(current_setting('cypra.tenant_id', true), '')::uuid);

ALTER TABLE upstream_providers ENABLE ROW LEVEL SECURITY;
CREATE POLICY tenant_isolation ON upstream_providers USING (tenant_id = NULLIF(current_setting('cypra.tenant_id', true), '')::uuid);

ALTER TABLE email_provider_configs ENABLE ROW LEVEL SECURITY;
CREATE POLICY tenant_isolation ON email_provider_configs USING (tenant_id = NULLIF(current_setting('cypra.tenant_id', true), '')::uuid);

ALTER TABLE email_outbox ENABLE ROW LEVEL SECURITY;
CREATE POLICY tenant_isolation ON email_outbox USING (tenant_id IS NULL OR tenant_id = NULLIF(current_setting('cypra.tenant_id', true), '')::uuid);

ALTER TABLE storage_objects ENABLE ROW LEVEL SECURITY;
CREATE POLICY tenant_isolation ON storage_objects USING (tenant_id = NULLIF(current_setting('cypra.tenant_id', true), '')::uuid);

ALTER TABLE tenant_auth_methods ENABLE ROW LEVEL SECURITY;
CREATE POLICY tenant_isolation ON tenant_auth_methods USING (tenant_id = NULLIF(current_setting('cypra.tenant_id', true), '')::uuid);

ALTER TABLE audit_entries ENABLE ROW LEVEL SECURITY;
CREATE POLICY tenant_isolation ON audit_entries USING (tenant_id IS NULL OR tenant_id = NULLIF(current_setting('cypra.tenant_id', true), '')::uuid);

ALTER TABLE rate_limit_buckets ENABLE ROW LEVEL SECURITY;
CREATE POLICY tenant_isolation ON rate_limit_buckets USING (tenant_id IS NULL OR tenant_id = NULLIF(current_setting('cypra.tenant_id', true), '')::uuid);

ALTER TABLE gdpr_deletions ENABLE ROW LEVEL SECURITY;
CREATE POLICY tenant_isolation ON gdpr_deletions USING (tenant_id = NULLIF(current_setting('cypra.tenant_id', true), '')::uuid);
