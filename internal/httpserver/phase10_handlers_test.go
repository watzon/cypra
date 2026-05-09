package httpserver_test

import (
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	cypra "github.com/watzon/cypra/internal/crypto"
	"github.com/watzon/cypra/internal/dbtest"
	"github.com/watzon/cypra/internal/httpserver"
)

func TestPhase10ProviderDiagnosticsMetricsAndTracing(t *testing.T) {
	t.Setenv("OTEL_EXPORTER_OTLP_ENDPOINT", "http://otel.example/v1/traces")
	t.Setenv("STORAGE_BACKEND", "s3-compatible")
	t.Setenv("STORAGE_S3_BUCKET", "cypra-test")
	t.Setenv("STORAGE_S3_ENDPOINT", "https://account.r2.cloudflarestorage.com")
	t.Setenv("STORAGE_S3_REGION", "auto")
	t.Setenv("STORAGE_S3_ACCESS_KEY_ID", "present")
	t.Setenv("STORAGE_S3_SECRET_ACCESS_KEY", "present")
	harness := dbtest.New(t)
	tenantID := dbtest.SeedTenant(t, harness.SQL, "acme")
	if _, err := harness.SQL.Exec(`INSERT INTO master_key_rotations (phase, rows_done, rows_total) VALUES ('rewrap', 3, 9)`); err != nil {
		t.Fatalf("seed rotation: %v", err)
	}
	if _, err := harness.SQL.Exec(`INSERT INTO email_outbox (tenant_id, to_address, template, payload) VALUES ($1, 'user@example.com', 'magic_link', '{}'::jsonb)`, tenantID); err != nil {
		t.Fatalf("seed email outbox: %v", err)
	}
	if _, err := harness.SQL.Exec(`INSERT INTO audit_entries (actor_kind, action, resource_kind) VALUES ('instance_admin', 'tenant.create', 'tenant')`); err != nil {
		t.Fatalf("seed audit entry: %v", err)
	}
	if _, err := harness.SQL.Exec(`INSERT INTO tenant_auth_methods (tenant_id, method, enabled, enrolled_count_cache) VALUES ($1, 'passkey', true, 2)`, tenantID); err != nil {
		t.Fatalf("seed auth method: %v", err)
	}
	if _, err := harness.SQL.Exec(`DELETE FROM schema_migrations WHERE version = '0009_instance_admin_webauthn'`); err != nil {
		t.Fatalf("mark migration pending: %v", err)
	}
	router := newTestServer(t, harness).Router()

	req := httptest.NewRequest(http.MethodGet, "/api/v1/provider-config/email", nil)
	req.Host = "acme.cypra.localhost"
	req.Header.Set("X-Cypra-Tenant-Role", "admin")
	resp := httptest.NewRecorder()
	router.ServeHTTP(resp, req)
	if resp.Code != http.StatusOK || !strings.Contains(resp.Body.String(), "Resend") || resp.Header().Get("Server-Timing") == "" {
		t.Fatalf("email provider response = %d headers=%v body=%s", resp.Code, resp.Header(), resp.Body.String())
	}

	req = httptest.NewRequest(http.MethodGet, "/api/v1/instance/diagnostics", nil)
	req.Host = "cypra.localhost"
	req.Header.Set("X-Cypra-Instance-Admin", "true")
	resp = httptest.NewRecorder()
	router.ServeHTTP(resp, req)
	if resp.Code != http.StatusOK || !strings.Contains(resp.Body.String(), "rewrap") || !strings.Contains(resp.Body.String(), "0009_instance_admin_webauthn") || !strings.Contains(resp.Body.String(), "https://****.cloudflarestorage.com") {
		t.Fatalf("diagnostics response = %d %s", resp.Code, resp.Body.String())
	}

	req = httptest.NewRequest(http.MethodGet, "/api/v1/instance/summary", nil)
	req.Host = "cypra.localhost"
	req.Header.Set("X-Cypra-Instance-Admin", "true")
	resp = httptest.NewRecorder()
	router.ServeHTTP(resp, req)
	if resp.Code != http.StatusOK || !strings.Contains(resp.Body.String(), `"tenants":1`) || !strings.Contains(resp.Body.String(), "tenant.create") {
		t.Fatalf("summary response = %d %s", resp.Code, resp.Body.String())
	}

	req = httptest.NewRequest(http.MethodGet, "/api/v1/auth-providers/", nil)
	req.Host = "acme.cypra.localhost"
	req.Header.Set("X-Cypra-Tenant-Role", "admin")
	resp = httptest.NewRecorder()
	router.ServeHTTP(resp, req)
	if resp.Code != http.StatusOK || !strings.Contains(resp.Body.String(), `"method":"passkey"`) || !strings.Contains(resp.Body.String(), `"enrolled_count":2`) {
		t.Fatalf("auth providers response = %d %s", resp.Code, resp.Body.String())
	}
	req = httptest.NewRequest(http.MethodPut, "/api/v1/auth-providers/passkey", strings.NewReader(`{"enabled":false,"config":{}}`))
	req.Host = "acme.cypra.localhost"
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Cypra-Tenant-Role", "admin")
	resp = httptest.NewRecorder()
	router.ServeHTTP(resp, req)
	if resp.Code != http.StatusOK || !strings.Contains(resp.Body.String(), `"enabled":false`) {
		t.Fatalf("save auth provider response = %d %s", resp.Code, resp.Body.String())
	}
	dbtest.RequireCount(t, harness.SQL, `SELECT count(*) FROM tenant_auth_methods WHERE tenant_id = $1 AND method = 'passkey' AND enabled = false`, 1, tenantID)

	req = httptest.NewRequest(http.MethodPut, "/api/v1/provider-config/email", strings.NewReader(`{"kind":"resend","from_address":"auth@example.com","from_name":"Cypra Auth","config":"re_test"}`))
	req.Host = "acme.cypra.localhost"
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Cypra-Tenant-Role", "admin")
	resp = httptest.NewRecorder()
	router.ServeHTTP(resp, req)
	if resp.Code != http.StatusOK || !strings.Contains(resp.Body.String(), "resend") {
		t.Fatalf("save email provider response = %d %s", resp.Code, resp.Body.String())
	}
	var storedEmailConfig []byte
	if err := harness.SQL.QueryRow(`SELECT config_encrypted FROM email_provider_configs WHERE tenant_id = $1`, tenantID).Scan(&storedEmailConfig); err != nil {
		t.Fatalf("load stored email config: %v", err)
	}
	if bytes.Equal(storedEmailConfig, []byte("re_test")) || len(storedEmailConfig) <= len("re_test") {
		t.Fatalf("email provider config was not encrypted: %q", storedEmailConfig)
	}
	req = httptest.NewRequest(http.MethodPost, "/api/v1/provider-config/email/test", nil)
	req.Host = "acme.cypra.localhost"
	req.Header.Set("X-Cypra-Tenant-Role", "admin")
	resp = httptest.NewRecorder()
	router.ServeHTTP(resp, req)
	if resp.Code != http.StatusOK || !strings.Contains(resp.Body.String(), `"healthy":true`) {
		t.Fatalf("test email provider response = %d %s", resp.Code, resp.Body.String())
	}

	req = httptest.NewRequest(http.MethodPut, "/api/v1/provider-config/upstream", strings.NewReader(`{"client_id":"google-client","client_secret":"google-secret","enabled":true}`))
	req.Host = "acme.cypra.localhost"
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Cypra-Tenant-Role", "admin")
	resp = httptest.NewRecorder()
	router.ServeHTTP(resp, req)
	if resp.Code != http.StatusOK || !strings.Contains(resp.Body.String(), "Google credentials decrypted successfully") {
		t.Fatalf("save upstream provider response = %d %s", resp.Code, resp.Body.String())
	}
	var storedClientID, storedClientSecret []byte
	if err := harness.SQL.QueryRow(`SELECT client_id_encrypted, client_secret_encrypted FROM upstream_providers WHERE tenant_id = $1`, tenantID).Scan(&storedClientID, &storedClientSecret); err != nil {
		t.Fatalf("load stored upstream provider: %v", err)
	}
	if bytes.Equal(storedClientID, []byte("google-client")) || bytes.Equal(storedClientSecret, []byte("google-secret")) {
		t.Fatalf("upstream provider secrets were not encrypted")
	}
	clientID, err := cypra.Decrypt(storedClientID[60:], storedClientID[:60], bytes.Repeat([]byte{7}, 32))
	if err != nil || string(clientID) != "google-client" {
		t.Fatalf("decrypt upstream client id = %q, %v", clientID, err)
	}
	clientSecret, err := cypra.Decrypt(storedClientSecret[60:], storedClientSecret[:60], bytes.Repeat([]byte{7}, 32))
	if err != nil || string(clientSecret) != "google-secret" {
		t.Fatalf("decrypt upstream client secret = %q, %v", clientSecret, err)
	}
	req = httptest.NewRequest(http.MethodPost, "/api/v1/provider-config/upstream/test", nil)
	req.Host = "acme.cypra.localhost"
	req.Header.Set("X-Cypra-Tenant-Role", "admin")
	resp = httptest.NewRecorder()
	router.ServeHTTP(resp, req)
	if resp.Code != http.StatusOK || !strings.Contains(resp.Body.String(), `"healthy":true`) {
		t.Fatalf("test upstream provider response = %d %s", resp.Code, resp.Body.String())
	}

	req = httptest.NewRequest(http.MethodGet, "/metrics", nil)
	req.Host = "cypra.localhost"
	resp = httptest.NewRecorder()
	router.ServeHTTP(resp, req)
	for _, metric := range []string{"cypra_http_request_duration_seconds", `cypra_master_key_rotation_phase{phase="rewrap"} 1`, "cypra_botmitigation_check_total", "cypra_migrations_pending 1", "cypra_email_outbox_pending 1", "cypra_signing_key_age_seconds"} {
		if !strings.Contains(resp.Body.String(), metric) {
			t.Fatalf("metrics missing %s: %s", metric, resp.Body.String())
		}
	}
}

func TestPhase10GDPRUserDeleteAndTenantCascade(t *testing.T) {
	harness := dbtest.New(t)
	tenantID := dbtest.SeedTenant(t, harness.SQL, "acme")
	userID := uuid.New()
	objectID := uuid.New()
	ownedObjectID := uuid.New()
	if _, err := harness.SQL.Exec(`INSERT INTO storage_objects (id, tenant_id, backend, key, content_type, byte_size) VALUES ($1, $2, 'local-disk', 'avatars/ada.png', 'image/png', 42)`, objectID, tenantID); err != nil {
		t.Fatalf("seed storage object: %v", err)
	}
	if _, err := harness.SQL.Exec(`INSERT INTO users (id, tenant_id, email, metadata, profile_picture_object_id) VALUES ($1, $2, 'ada@example.com', '{"name":"Ada"}'::jsonb, $3)`, userID, tenantID, objectID); err != nil {
		t.Fatalf("seed user: %v", err)
	}
	if _, err := harness.SQL.Exec(`INSERT INTO storage_objects (id, tenant_id, backend, key, content_type, byte_size, owner_user_id) VALUES ($1, $2, 'local-disk', 'exports/ada.json', 'application/json', 128, $3)`, ownedObjectID, tenantID, userID); err != nil {
		t.Fatalf("seed owned storage object: %v", err)
	}
	if _, err := harness.SQL.Exec(`INSERT INTO audit_entries (tenant_id, actor_kind, action, resource_kind, resource_id, state_after, metadata) VALUES ($1, 'tenant_admin', 'user.update', 'user', $2, '{"email":"ada@example.com"}'::jsonb, '{}'::jsonb)`, tenantID, userID); err != nil {
		t.Fatalf("seed audit: %v", err)
	}
	projectID := uuid.New()
	clientID := uuid.New()
	if _, err := harness.SQL.Exec(`INSERT INTO projects (id, tenant_id, slug, name) VALUES ($1, $2, 'app', 'App')`, projectID, tenantID); err != nil {
		t.Fatalf("seed project: %v", err)
	}
	if _, err := harness.SQL.Exec(`INSERT INTO oidc_clients (id, tenant_id, project_id, client_id, redirect_uris, allowed_scopes, token_endpoint_auth_method) VALUES ($1, $2, $3, 'client', ARRAY['https://app.example/cb'], ARRAY['openid'], 'none')`, clientID, tenantID, projectID); err != nil {
		t.Fatalf("seed client: %v", err)
	}
	if _, err := harness.SQL.Exec(`INSERT INTO oidc_refresh_tokens (family_id, token_hash, tenant_id, oidc_client_uuid, user_id, scope, expires_at) VALUES ($1, 'hash'::bytea, $2, $3, $4, ARRAY['openid'], now() + interval '1 hour')`, uuid.New(), tenantID, clientID, userID); err != nil {
		t.Fatalf("seed refresh: %v", err)
	}
	router := newTestServer(t, harness).Router()

	req := httptest.NewRequest(http.MethodGet, "/api/v1/users/"+userID.String()+"/export", nil)
	req.Host = "acme.cypra.localhost"
	req.Header.Set("X-Cypra-Tenant-Role", "admin")
	resp := httptest.NewRecorder()
	router.ServeHTTP(resp, req)
	if resp.Code != http.StatusOK || !strings.Contains(resp.Body.String(), "ada@example.com") || !strings.Contains(resp.Body.String(), "exports/ada.json") {
		t.Fatalf("export response = %d %s", resp.Code, resp.Body.String())
	}

	req = httptest.NewRequest(http.MethodDelete, "/api/v1/users/"+userID.String(), nil)
	req.Host = "acme.cypra.localhost"
	req.Header.Set("X-Cypra-Tenant-Role", "admin")
	resp = httptest.NewRecorder()
	router.ServeHTTP(resp, req)
	if resp.Code != http.StatusOK {
		t.Fatalf("delete user response = %d %s", resp.Code, resp.Body.String())
	}
	dbtest.RequireCount(t, harness.SQL, `SELECT count(*) FROM users WHERE email LIKE '*** redacted%'`, 1)
	dbtest.RequireCount(t, harness.SQL, `SELECT count(*) FROM gdpr_deletions WHERE state = 'done'`, 1)
	dbtest.RequireCount(t, harness.SQL, `SELECT count(*) FROM audit_entries WHERE redacted_at IS NOT NULL`, 1)
	dbtest.RequireCount(t, harness.SQL, `SELECT count(*) FROM storage_objects WHERE deleted_at IS NOT NULL`, 2)

	req = httptest.NewRequest(http.MethodPost, "/api/v1/tenants/"+tenantID.String()+"/delete", bytes.NewReader([]byte(`{"slug":"acme"}`)))
	req.Host = "cypra.localhost"
	req.Header.Set("X-Cypra-Instance-Admin", "true")
	resp = httptest.NewRecorder()
	router.ServeHTTP(resp, req)
	if resp.Code != http.StatusOK {
		t.Fatalf("delete tenant response = %d %s", resp.Code, resp.Body.String())
	}
	dbtest.RequireCount(t, harness.SQL, `SELECT count(*) FROM oidc_refresh_tokens WHERE revoked_at IS NOT NULL`, 1)
	dbtest.RequireCount(t, harness.SQL, `SELECT count(*) FROM tenants WHERE id = $1 AND settings ? 'deletion_scheduled_at'`, 1, tenantID)
	dbtest.RequireCount(t, harness.SQL, `SELECT count(*) FROM oidc_signing_keys WHERE state = 'sunsetting' AND sunset_until IS NOT NULL`, 1)

	req = httptest.NewRequest(http.MethodGet, "https://acme.cypra.localhost/.well-known/jwks.json", nil)
	resp = httptest.NewRecorder()
	router.ServeHTTP(resp, req)
	if resp.Code != http.StatusOK || !strings.Contains(resp.Body.String(), "keys") {
		t.Fatalf("jwks during sunset = %d %s", resp.Code, resp.Body.String())
	}
	if _, err := harness.SQL.Exec(`UPDATE tenants SET settings = settings || jsonb_build_object('hard_delete_after', now() - interval '1 second') WHERE id = $1`, tenantID); err != nil {
		t.Fatalf("expire tenant deletion: %v", err)
	}
	if _, err := harness.SQL.Exec(`UPDATE oidc_signing_keys SET sunset_until = now() - interval '1 second' WHERE tenant_id = $1`, tenantID); err != nil {
		t.Fatalf("expire signing keys: %v", err)
	}
	if err := httpserver.RunTenantDeletionCleanup(context.Background(), harness.SQL, time.Now().Add(time.Second)); err != nil {
		t.Fatalf("tenant deletion cleanup: %v", err)
	}
	dbtest.RequireCount(t, harness.SQL, `SELECT count(*) FROM tenants WHERE id = $1`, 0, tenantID)
	dbtest.RequireCount(t, harness.SQL, `SELECT count(*) FROM oidc_signing_keys WHERE tenant_id = $1`, 0, tenantID)
}

func TestPhase10InstanceAdminsListInviteDemoteAndLastAdminGuard(t *testing.T) {
	harness := dbtest.New(t)
	firstID := uuid.New()
	secondID := uuid.New()
	if _, err := harness.SQL.Exec(`INSERT INTO instance_admins (id, email, display_name) VALUES ($1, 'first@example.com', 'First'), ($2, 'second@example.com', 'Second')`, firstID, secondID); err != nil {
		t.Fatalf("seed admins: %v", err)
	}
	router := newTestServer(t, harness).Router()

	req := httptest.NewRequest(http.MethodGet, "/api/v1/instance/admins", nil)
	req.Host = "cypra.localhost"
	req.Header.Set("X-Cypra-Instance-Admin", "true")
	resp := httptest.NewRecorder()
	router.ServeHTTP(resp, req)
	if resp.Code != http.StatusOK || !strings.Contains(resp.Body.String(), "first@example.com") || !strings.Contains(resp.Body.String(), "second@example.com") {
		t.Fatalf("list admins response = %d %s", resp.Code, resp.Body.String())
	}

	req = httptest.NewRequest(http.MethodPost, "/api/v1/instance/invite", strings.NewReader(`{"email":"third@example.com","redirect_url":"https://cypra.localhost/setup"}`))
	req.Host = "cypra.localhost"
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Cypra-Instance-Admin", "true")
	resp = httptest.NewRecorder()
	router.ServeHTTP(resp, req)
	if resp.Code != http.StatusCreated || !strings.Contains(resp.Body.String(), "token") {
		t.Fatalf("invite admin response = %d %s", resp.Code, resp.Body.String())
	}
	var inviteID string
	if err := harness.SQL.QueryRow(`SELECT id FROM pending_invitations WHERE tenant_id IS NULL AND email = 'third@example.com'`).Scan(&inviteID); err != nil {
		t.Fatalf("load pending invite: %v", err)
	}
	req = httptest.NewRequest(http.MethodGet, "/api/v1/instance/invites", nil)
	req.Host = "cypra.localhost"
	req.Header.Set("X-Cypra-Instance-Admin", "true")
	resp = httptest.NewRecorder()
	router.ServeHTTP(resp, req)
	if resp.Code != http.StatusOK || !strings.Contains(resp.Body.String(), "third@example.com") {
		t.Fatalf("list invites response = %d %s", resp.Code, resp.Body.String())
	}
	req = httptest.NewRequest(http.MethodDelete, "/api/v1/instance/invites/"+inviteID, nil)
	req.Host = "cypra.localhost"
	req.Header.Set("X-Cypra-Instance-Admin", "true")
	resp = httptest.NewRecorder()
	router.ServeHTTP(resp, req)
	if resp.Code != http.StatusOK {
		t.Fatalf("revoke invite response = %d %s", resp.Code, resp.Body.String())
	}
	dbtest.RequireCount(t, harness.SQL, `SELECT count(*) FROM pending_invitations WHERE id = $1`, 0, inviteID)

	req = httptest.NewRequest(http.MethodDelete, "/api/v1/instance/admins/"+secondID.String(), nil)
	req.Host = "cypra.localhost"
	req.Header.Set("X-Cypra-Instance-Admin", "true")
	resp = httptest.NewRecorder()
	router.ServeHTTP(resp, req)
	if resp.Code != http.StatusOK {
		t.Fatalf("demote admin response = %d %s", resp.Code, resp.Body.String())
	}
	dbtest.RequireCount(t, harness.SQL, `SELECT count(*) FROM instance_admins WHERE disabled_at IS NULL`, 1)

	req = httptest.NewRequest(http.MethodDelete, "/api/v1/instance/admins/"+firstID.String(), nil)
	req.Host = "cypra.localhost"
	req.Header.Set("X-Cypra-Instance-Admin", "true")
	resp = httptest.NewRecorder()
	router.ServeHTTP(resp, req)
	if resp.Code != http.StatusConflict || !strings.Contains(resp.Body.String(), "instance_admin.last_admin") {
		t.Fatalf("last admin response = %d %s", resp.Code, resp.Body.String())
	}
}
