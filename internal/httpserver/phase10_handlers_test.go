package httpserver_test

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/watzon/cypra/internal/dbtest"
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
	dbtest.SeedTenant(t, harness.SQL, "acme")
	if _, err := harness.SQL.Exec(`INSERT INTO master_key_rotations (phase, rows_done, rows_total) VALUES ('rewrap', 3, 9)`); err != nil {
		t.Fatalf("seed rotation: %v", err)
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
	if resp.Code != http.StatusOK || !strings.Contains(resp.Body.String(), "rewrap") || !strings.Contains(resp.Body.String(), "https://****.cloudflarestorage.com") {
		t.Fatalf("diagnostics response = %d %s", resp.Code, resp.Body.String())
	}

	req = httptest.NewRequest(http.MethodPut, "/api/v1/provider-config/email", strings.NewReader(`{"kind":"resend","from_address":"auth@example.com","from_name":"Cypra Auth","config":"re_test"}`))
	req.Host = "acme.cypra.localhost"
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Cypra-Tenant-Role", "admin")
	resp = httptest.NewRecorder()
	router.ServeHTTP(resp, req)
	if resp.Code != http.StatusOK || !strings.Contains(resp.Body.String(), "resend") {
		t.Fatalf("save email provider response = %d %s", resp.Code, resp.Body.String())
	}

	req = httptest.NewRequest(http.MethodPut, "/api/v1/provider-config/upstream", strings.NewReader(`{"client_id":"google-client","client_secret":"google-secret","enabled":true}`))
	req.Host = "acme.cypra.localhost"
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Cypra-Tenant-Role", "admin")
	resp = httptest.NewRecorder()
	router.ServeHTTP(resp, req)
	if resp.Code != http.StatusOK || !strings.Contains(resp.Body.String(), "Google upstream configured") {
		t.Fatalf("save upstream provider response = %d %s", resp.Code, resp.Body.String())
	}

	req = httptest.NewRequest(http.MethodGet, "/metrics", nil)
	req.Host = "cypra.localhost"
	resp = httptest.NewRecorder()
	router.ServeHTTP(resp, req)
	for _, metric := range []string{"cypra_http_request_duration_seconds", "cypra_master_key_rotation_phase", "cypra_botmitigation_check_total"} {
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
	if _, err := harness.SQL.Exec(`INSERT INTO storage_objects (id, tenant_id, backend, key, content_type, byte_size) VALUES ($1, $2, 'local-disk', 'avatars/ada.png', 'image/png', 42)`, objectID, tenantID); err != nil {
		t.Fatalf("seed storage object: %v", err)
	}
	if _, err := harness.SQL.Exec(`INSERT INTO users (id, tenant_id, email, metadata, profile_picture_object_id) VALUES ($1, $2, 'ada@example.com', '{"name":"Ada"}'::jsonb, $3)`, userID, tenantID, objectID); err != nil {
		t.Fatalf("seed user: %v", err)
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
	if resp.Code != http.StatusOK || !strings.Contains(resp.Body.String(), "ada@example.com") {
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
	dbtest.RequireCount(t, harness.SQL, `SELECT count(*) FROM storage_objects WHERE deleted_at IS NOT NULL`, 1)

	req = httptest.NewRequest(http.MethodDelete, "/api/v1/tenants/"+tenantID.String(), bytes.NewReader(nil))
	req.Host = "cypra.localhost"
	req.Header.Set("X-Cypra-Instance-Admin", "true")
	resp = httptest.NewRecorder()
	router.ServeHTTP(resp, req)
	if resp.Code != http.StatusNoContent {
		t.Fatalf("delete tenant response = %d %s", resp.Code, resp.Body.String())
	}
	dbtest.RequireCount(t, harness.SQL, `SELECT count(*) FROM oidc_refresh_tokens WHERE revoked_at IS NOT NULL`, 1)
	dbtest.RequireCount(t, harness.SQL, `SELECT count(*) FROM oidc_clients WHERE deleted_at IS NOT NULL`, 1)
	dbtest.RequireCount(t, harness.SQL, `SELECT count(*) FROM oidc_signing_keys WHERE state = 'sunsetting' AND sunset_until IS NOT NULL`, 1)
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
