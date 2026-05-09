package httpserver_test

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/watzon/cypra/internal/dbtest"
	"github.com/watzon/cypra/internal/httpserver"
	"github.com/watzon/cypra/internal/logging"
)

func TestHealthReadyAndTenantCRUD(t *testing.T) {
	harness := dbtest.New(t)
	router := newTestServer(t, harness).Router()

	req := httptest.NewRequest(http.MethodGet, "/healthz", nil)
	resp := httptest.NewRecorder()
	router.ServeHTTP(resp, req)
	if resp.Code != http.StatusOK {
		t.Fatalf("healthz status = %d", resp.Code)
	}

	req = httptest.NewRequest(http.MethodGet, "/readyz", nil)
	resp = httptest.NewRecorder()
	router.ServeHTTP(resp, req)
	if resp.Code != http.StatusOK {
		t.Fatalf("readyz status = %d body=%s", resp.Code, resp.Body.String())
	}

	req = httptest.NewRequest(http.MethodPost, "/api/v1/tenants/", bytes.NewReader([]byte(`{"slug":"acme","name":"Acme"}`)))
	req.Host = "cypra.localhost"
	req.Header.Set("X-Cypra-Instance-Admin", "true")
	resp = httptest.NewRecorder()
	router.ServeHTTP(resp, req)
	if resp.Code != http.StatusCreated {
		t.Fatalf("create tenant status = %d body=%s", resp.Code, resp.Body.String())
	}
	var tenant map[string]any
	if err := json.Unmarshal(resp.Body.Bytes(), &tenant); err != nil {
		t.Fatalf("decode tenant: %v", err)
	}
	if tenant["slug"] != "acme" {
		t.Fatalf("tenant slug = %#v", tenant["slug"])
	}

	req = httptest.NewRequest(http.MethodPost, "/api/v1/tenants/", bytes.NewReader([]byte(`{"slug":"admin","name":"Admin"}`)))
	req.Host = "cypra.localhost"
	req.Header.Set("X-Cypra-Instance-Admin", "true")
	resp = httptest.NewRecorder()
	router.ServeHTTP(resp, req)
	if resp.Code != http.StatusBadRequest || !bytes.Contains(resp.Body.Bytes(), []byte("tenant.slug_reserved")) {
		t.Fatalf("reserved slug response = %d %s", resp.Code, resp.Body.String())
	}

	req = httptest.NewRequest(http.MethodPost, "/api/v1/tenants/", bytes.NewReader([]byte(`{"slug":"1bad","name":"Bad"}`)))
	req.Host = "cypra.localhost"
	req.Header.Set("X-Cypra-Instance-Admin", "true")
	resp = httptest.NewRecorder()
	router.ServeHTTP(resp, req)
	if resp.Code != http.StatusBadRequest || !bytes.Contains(resp.Body.Bytes(), []byte("tenant.slug_invalid")) {
		t.Fatalf("invalid slug response = %d %s", resp.Code, resp.Body.String())
	}

	req = httptest.NewRequest(http.MethodPost, "/api/v1/tenants/", bytes.NewReader([]byte(`{"slug":"bravo","name":"Bravo"}`)))
	req.Host = "cypra.localhost"
	resp = httptest.NewRecorder()
	router.ServeHTTP(resp, req)
	if resp.Code != http.StatusForbidden {
		t.Fatalf("non-admin tenant create status = %d", resp.Code)
	}

	req = httptest.NewRequest(http.MethodGet, "/api/v1/openapi.json", nil)
	req.Host = "cypra.localhost"
	resp = httptest.NewRecorder()
	router.ServeHTTP(resp, req)
	if resp.Code != http.StatusOK || !bytes.Contains(resp.Body.Bytes(), []byte("/api/v1/tenants")) {
		t.Fatalf("openapi response = %d %s", resp.Code, resp.Body.String())
	}
}

func TestBotVerifierFailureBlocksAuthHandler(t *testing.T) {
	harness := dbtest.New(t)
	dbtest.SeedTenant(t, harness.SQL, "acme")
	server, err := httpserver.New(httpserver.Options{DB: harness.SQL, TenantDB: harness.TenantDB, PublicBaseURL: "https://cypra.localhost", Version: "test", Commit: "test", KEKLoaded: true, BotVerifier: failingVerifier{}, TrustDevHeaders: true})
	if err != nil {
		t.Fatalf("new server: %v", err)
	}
	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/password/signup", bytes.NewReader([]byte(`{"email":"user@example.com","password":"correct horse battery staple","bot_token":"bad"}`)))
	req.Host = "acme.cypra.localhost"
	resp := httptest.NewRecorder()
	server.Router().ServeHTTP(resp, req)
	if resp.Code != http.StatusForbidden || !bytes.Contains(resp.Body.Bytes(), []byte("bot.verify_failed")) {
		t.Fatalf("bot failure response = %d %s", resp.Code, resp.Body.String())
	}
	dbtest.RequireCount(t, harness.SQL, `SELECT count(*) FROM users WHERE email = 'user@example.com'`, 0)
	dbtest.RequireCount(t, harness.SQL, `SELECT count(*) FROM audit_entries WHERE action = 'bot.verify_failed' AND resource_kind = 'bot_mitigation'`, 1)
}

func TestAuthAndOIDCRoutesReturnRateLimitedAtThresholds(t *testing.T) {
	harness := dbtest.New(t)
	dbtest.SeedTenant(t, harness.SQL, "acme")
	router := newTestServer(t, harness).Router()

	tests := []struct {
		name        string
		method      string
		path        string
		body        func() string
		contentType string
		attempts    int
	}{
		{
			name:   "password signup",
			method: http.MethodPost,
			path:   "/api/v1/auth/password/signup",
			body: func() string {
				return `{"email":"limited-signup@example.com","password":"correct horse battery staple"}`
			},
			attempts: 6,
		},
		{
			name:     "password login",
			method:   http.MethodPost,
			path:     "/api/v1/auth/password/signin",
			body:     func() string { return `{"email":"limited-login@example.com","password":"wrong password"}` },
			attempts: 6,
		},
		{
			name:   "password reset",
			method: http.MethodPost,
			path:   "/api/v1/auth/password/reset",
			body: func() string {
				return `{"user_id":"limited-reset@example.com","password":"correct horse battery staple"}`
			},
			attempts: 6,
		},
		{
			name:     "magic link issue",
			method:   http.MethodPost,
			path:     "/api/v1/auth/magic-link/issue",
			body:     func() string { return `{"email":"limited-magic@example.com"}` },
			attempts: 6,
		},
		{
			name:     "invite redeem",
			method:   http.MethodPost,
			path:     "/api/v1/auth/invite/redeem",
			body:     func() string { return `{"token":"limited-invite-token"}` },
			attempts: 6,
		},
		{
			name:   "oidc token",
			method: http.MethodPost,
			path:   "/oidc/token",
			body: func() string {
				return url.Values{"grant_type": {"authorization_code"}, "client_id": {"limited-client"}, "code": {"invalid-code"}}.Encode()
			},
			contentType: "application/x-www-form-urlencoded",
			attempts:    61,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var resp *httptest.ResponseRecorder
			for i := 0; i < tt.attempts; i++ {
				req := httptest.NewRequest(tt.method, tt.path, strings.NewReader(tt.body()))
				req.Host = "acme.cypra.localhost"
				if tt.contentType != "" {
					req.Header.Set("Content-Type", tt.contentType)
				}
				resp = httptest.NewRecorder()
				router.ServeHTTP(resp, req)
			}
			if resp.Code != http.StatusTooManyRequests || !bytes.Contains(resp.Body.Bytes(), []byte("auth.rate_limited")) {
				t.Fatalf("final response = %d %s", resp.Code, resp.Body.String())
			}
		})
	}
}

func TestListUsersSupportsPagedLimit(t *testing.T) {
	harness := dbtest.New(t)
	tenantID := dbtest.SeedTenant(t, harness.SQL, "acme")
	for i := range 3 {
		if _, err := harness.SQL.Exec(`INSERT INTO users (id, tenant_id, email, metadata) VALUES ($1, $2, $3, '{}'::jsonb)`, uuid.New(), tenantID, "user-0"+string(rune('1'+i))+"@example.com"); err != nil {
			t.Fatalf("seed user: %v", err)
		}
	}
	router := newTestServer(t, harness).Router()

	req := httptest.NewRequest(http.MethodGet, "/api/v1/users/?limit=2&page=2", nil)
	req.Host = "acme.cypra.localhost"
	req.Header.Set("X-Cypra-Tenant-Role", "admin")
	resp := httptest.NewRecorder()
	router.ServeHTTP(resp, req)
	if resp.Code != http.StatusOK {
		t.Fatalf("list users status = %d body=%s", resp.Code, resp.Body.String())
	}
	var users []map[string]any
	if err := json.Unmarshal(resp.Body.Bytes(), &users); err != nil {
		t.Fatalf("decode users: %v", err)
	}
	if len(users) != 1 || users[0]["email"] != "user-03@example.com" {
		t.Fatalf("paged users = %#v", users)
	}
}

func TestHTTPLogsRequiredFieldsAndRedactsRequestPII(t *testing.T) {
	harness := dbtest.New(t)
	tenantID := dbtest.SeedTenant(t, harness.SQL, "acme")
	var logs bytes.Buffer
	server, err := httpserver.New(httpserver.Options{DB: harness.SQL, TenantDB: harness.TenantDB, PublicBaseURL: "https://cypra.localhost", Version: "test", Commit: "test", KEKLoaded: true, Logger: slog.New(logging.RedactingHandler{Handler: slog.NewTextHandler(&logs, nil)}), TrustDevHeaders: true})
	if err != nil {
		t.Fatalf("new server: %v", err)
	}
	router := server.Router()

	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/password/signin", strings.NewReader(`{"email":"ada@example.com","password":"secret-password"}`))
	req.Host = "acme.cypra.localhost"
	req.Header.Set("X-Request-Id", "req-auth-failure")
	resp := httptest.NewRecorder()
	router.ServeHTTP(resp, req)
	if resp.Code != http.StatusUnauthorized {
		t.Fatalf("auth failure status = %d %s", resp.Code, resp.Body.String())
	}

	actorID := "11111111-1111-1111-1111-111111111111"
	req = httptest.NewRequest(http.MethodGet, "/api/v1/projects/?email=ada@example.com&token=secret-token", nil)
	req.Host = "acme.cypra.localhost"
	req.Header.Set("X-Request-Id", "req-project-list")
	req.Header.Set("X-Cypra-Tenant-Role", "admin")
	req.Header.Set("X-Cypra-User-Id", actorID)
	resp = httptest.NewRecorder()
	router.ServeHTTP(resp, req)
	if resp.Code != http.StatusOK {
		t.Fatalf("project list status = %d %s", resp.Code, resp.Body.String())
	}

	logged := logs.String()
	for _, want := range []string{"request_id=req-auth-failure", "request_id=req-project-list", "tenant_id=" + tenantID.String(), "actor_id=" + actorID, "status=401", "status=200"} {
		if !strings.Contains(logged, want) {
			t.Fatalf("log missing %q: %s", want, logged)
		}
	}
	for _, leaked := range []string{"ada@example.com", "secret-password", "secret-token"} {
		if strings.Contains(logged, leaked) {
			t.Fatalf("log leaked %q: %s", leaked, logged)
		}
	}
}

func TestTenantResolverProjectCRUDAndAudit(t *testing.T) {
	harness := dbtest.New(t)
	dbtest.SeedTenant(t, harness.SQL, "acme")
	router := newTestServer(t, harness).Router()

	req := httptest.NewRequest(http.MethodPost, "/api/v1/projects/", bytes.NewReader([]byte(`{"slug":"app","name":"App"}`)))
	req.Host = "acme.cypra.localhost"
	req.Header.Set("X-Cypra-Tenant-Role", "admin")
	resp := httptest.NewRecorder()
	router.ServeHTTP(resp, req)
	if resp.Code != http.StatusCreated {
		t.Fatalf("create project status = %d body=%s", resp.Code, resp.Body.String())
	}
	var created map[string]any
	if err := json.Unmarshal(resp.Body.Bytes(), &created); err != nil {
		t.Fatalf("decode created project: %v", err)
	}
	projectID := created["id"].(string)
	if created["client_secret"] == "" || created["client_id"] == "" {
		t.Fatalf("created project missing oidc fields: %+v", created)
	}
	dbtest.RequireCount(t, harness.SQL, `SELECT count(*) FROM projects WHERE slug = 'app'`, 1)
	dbtest.RequireCount(t, harness.SQL, `SELECT count(*) FROM oidc_clients WHERE client_id = 'client_acme_app'`, 1)
	dbtest.RequireCount(t, harness.SQL, `SELECT count(*) FROM audit_entries WHERE action = 'project.create'`, 1)

	req = httptest.NewRequest(http.MethodPut, "/api/v1/projects/"+projectID, bytes.NewReader([]byte(`{"name":"App","redirect_uris":["https://app.example/callback"],"allowed_scopes":["openid","email"],"token_endpoint_auth_method":"none"}`)))
	req.Host = "acme.cypra.localhost"
	req.Header.Set("X-Cypra-Tenant-Role", "admin")
	resp = httptest.NewRecorder()
	router.ServeHTTP(resp, req)
	if resp.Code != http.StatusOK {
		t.Fatalf("update project status = %d body=%s", resp.Code, resp.Body.String())
	}
	dbtest.RequireCount(t, harness.SQL, `SELECT count(*) FROM oidc_clients WHERE project_id = $1 AND token_endpoint_auth_method = 'none'`, 1, projectID)

	req = httptest.NewRequest(http.MethodPost, "/api/v1/projects/"+projectID+"/rotate-secret", nil)
	req.Host = "acme.cypra.localhost"
	req.Header.Set("X-Cypra-Tenant-Role", "admin")
	resp = httptest.NewRecorder()
	router.ServeHTTP(resp, req)
	if resp.Code != http.StatusOK || !strings.Contains(resp.Body.String(), "client_secret") {
		t.Fatalf("rotate project secret status = %d body=%s", resp.Code, resp.Body.String())
	}

	req = httptest.NewRequest(http.MethodDelete, "/api/v1/projects/"+projectID, nil)
	req.Host = "acme.cypra.localhost"
	req.Header.Set("X-Cypra-Tenant-Role", "admin")
	resp = httptest.NewRecorder()
	router.ServeHTTP(resp, req)
	if resp.Code != http.StatusNoContent {
		t.Fatalf("delete project status = %d body=%s", resp.Code, resp.Body.String())
	}
	dbtest.RequireCount(t, harness.SQL, `SELECT count(*) FROM projects WHERE id = $1 AND deleted_at IS NOT NULL`, 1, projectID)

	req = httptest.NewRequest(http.MethodGet, "/api/v1/projects/", nil)
	req.Host = "unknown.cypra.localhost"
	req.Header.Set("X-Cypra-Tenant-Role", "admin")
	resp = httptest.NewRecorder()
	router.ServeHTTP(resp, req)
	if resp.Code != http.StatusNotFound {
		t.Fatalf("unknown tenant status = %d", resp.Code)
	}
}

func TestAuditExportRequiresTenantAdminAndStaysTenantScoped(t *testing.T) {
	harness := dbtest.New(t)
	acmeID := dbtest.SeedTenant(t, harness.SQL, "acme")
	bravoID := dbtest.SeedTenant(t, harness.SQL, "bravo")
	if _, err := harness.SQL.Exec(`INSERT INTO audit_entries (tenant_id, actor_kind, action, resource_kind) VALUES ($1, 'system', 'acme.only', 'tenant')`, acmeID); err != nil {
		t.Fatalf("seed acme audit: %v", err)
	}
	if _, err := harness.SQL.Exec(`INSERT INTO audit_entries (tenant_id, actor_kind, action, resource_kind) VALUES ($1, 'system', 'bravo.only', 'tenant')`, bravoID); err != nil {
		t.Fatalf("seed bravo audit: %v", err)
	}
	router := newTestServer(t, harness).Router()

	req := httptest.NewRequest(http.MethodGet, "https://acme.cypra.localhost/api/v1/audit/export", nil)
	resp := httptest.NewRecorder()
	router.ServeHTTP(resp, req)
	if resp.Code != http.StatusForbidden {
		t.Fatalf("unauthorized export status = %d", resp.Code)
	}

	req = httptest.NewRequest(http.MethodGet, "https://acme.cypra.localhost/api/v1/audit/export", nil)
	req.Header.Set("X-Cypra-Tenant-Role", "member")
	resp = httptest.NewRecorder()
	router.ServeHTTP(resp, req)
	if resp.Code != http.StatusForbidden {
		t.Fatalf("member export status = %d", resp.Code)
	}

	req = httptest.NewRequest(http.MethodGet, "https://acme.cypra.localhost/api/v1/audit/export", nil)
	req.Header.Set("X-Cypra-Tenant-Role", "admin")
	resp = httptest.NewRecorder()
	router.ServeHTTP(resp, req)
	if resp.Code != http.StatusOK {
		t.Fatalf("admin export status = %d body=%s", resp.Code, resp.Body.String())
	}
	body := resp.Body.String()
	if !bytes.Contains([]byte(body), []byte("acme.only")) || bytes.Contains([]byte(body), []byte("bravo.only")) {
		t.Fatalf("tenant-scoped export failed: %s", body)
	}

	req = httptest.NewRequest(http.MethodGet, "https://acme.cypra.localhost/api/v1/audit/?action=acme", nil)
	req.Header.Set("X-Cypra-Tenant-Role", "admin")
	resp = httptest.NewRecorder()
	router.ServeHTTP(resp, req)
	if resp.Code != http.StatusOK || !strings.Contains(resp.Body.String(), "acme.only") || strings.Contains(resp.Body.String(), "bravo.only") {
		t.Fatalf("tenant audit list = %d %s", resp.Code, resp.Body.String())
	}

	req = httptest.NewRequest(http.MethodGet, "https://cypra.localhost/api/v1/instance/audit?action=bravo", nil)
	req.Header.Set("X-Cypra-Instance-Admin", "true")
	resp = httptest.NewRecorder()
	router.ServeHTTP(resp, req)
	if resp.Code != http.StatusOK || !strings.Contains(resp.Body.String(), "bravo.only") || strings.Contains(resp.Body.String(), "acme.only") {
		t.Fatalf("instance audit list = %d %s", resp.Code, resp.Body.String())
	}
}

func TestTenantInviteListAndRevoke(t *testing.T) {
	harness := dbtest.New(t)
	tenantID := dbtest.SeedTenant(t, harness.SQL, "acme")
	inviteID := "00000000-0000-0000-0000-00000000aa01"
	if _, err := harness.SQL.Exec(`INSERT INTO pending_invitations (id, tenant_id, email, role, token_hash, created_by_kind, expires_at) VALUES ($1, $2, 'pending@example.com', 'member', 'hash'::bytea, 'tenant_admin', now() + interval '7 days')`, inviteID, tenantID); err != nil {
		t.Fatalf("seed invite: %v", err)
	}
	router := newTestServer(t, harness).Router()
	req := httptest.NewRequest(http.MethodGet, "https://acme.cypra.localhost/api/v1/admin/invites", nil)
	req.Header.Set("X-Cypra-Tenant-Role", "admin")
	resp := httptest.NewRecorder()
	router.ServeHTTP(resp, req)
	if resp.Code != http.StatusOK || !strings.Contains(resp.Body.String(), "pending@example.com") {
		t.Fatalf("list tenant invites = %d %s", resp.Code, resp.Body.String())
	}

	req = httptest.NewRequest(http.MethodDelete, "https://acme.cypra.localhost/api/v1/admin/invites/"+inviteID, nil)
	req.Header.Set("X-Cypra-Tenant-Role", "admin")
	resp = httptest.NewRecorder()
	router.ServeHTTP(resp, req)
	if resp.Code != http.StatusOK {
		t.Fatalf("revoke tenant invite = %d %s", resp.Code, resp.Body.String())
	}
	dbtest.RequireCount(t, harness.SQL, `SELECT count(*) FROM pending_invitations WHERE id = $1`, 0, inviteID)
}

func TestTenantMembersListUpdateRemoveAndLastOwnerGuard(t *testing.T) {
	harness := dbtest.New(t)
	tenantID := dbtest.SeedTenant(t, harness.SQL, "acme")
	ownerID := uuid.New()
	adminID := uuid.New()
	ownerMembershipID := uuid.New()
	adminMembershipID := uuid.New()
	if _, err := harness.SQL.Exec(`INSERT INTO users (id, tenant_id, email, metadata) VALUES ($1, $2, 'owner@example.com', '{}'::jsonb), ($3, $2, 'admin@example.com', '{}'::jsonb)`, ownerID, tenantID, adminID); err != nil {
		t.Fatalf("seed users: %v", err)
	}
	if _, err := harness.SQL.Exec(`INSERT INTO tenant_memberships (id, tenant_id, user_id, role) VALUES ($1, $2, $3, 'owner'), ($4, $2, $5, 'admin')`, ownerMembershipID, tenantID, ownerID, adminMembershipID, adminID); err != nil {
		t.Fatalf("seed memberships: %v", err)
	}
	router := newTestServer(t, harness).Router()

	req := httptest.NewRequest(http.MethodGet, "/api/v1/admin/members", nil)
	req.Host = "acme.cypra.localhost"
	req.Header.Set("X-Cypra-Tenant-Role", "admin")
	resp := httptest.NewRecorder()
	router.ServeHTTP(resp, req)
	if resp.Code != http.StatusOK || !strings.Contains(resp.Body.String(), "owner@example.com") || !strings.Contains(resp.Body.String(), "admin@example.com") {
		t.Fatalf("list members = %d %s", resp.Code, resp.Body.String())
	}

	req = httptest.NewRequest(http.MethodPut, "/api/v1/admin/members/"+adminMembershipID.String(), strings.NewReader(`{"role":"member"}`))
	req.Host = "acme.cypra.localhost"
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Cypra-Tenant-Role", "admin")
	resp = httptest.NewRecorder()
	router.ServeHTTP(resp, req)
	if resp.Code != http.StatusOK || !strings.Contains(resp.Body.String(), `"role":"member"`) {
		t.Fatalf("update member = %d %s", resp.Code, resp.Body.String())
	}
	dbtest.RequireCount(t, harness.SQL, `SELECT count(*) FROM tenant_memberships WHERE id = $1 AND role = 'member'`, 1, adminMembershipID)

	req = httptest.NewRequest(http.MethodDelete, "/api/v1/admin/members/"+ownerMembershipID.String(), nil)
	req.Host = "acme.cypra.localhost"
	req.Header.Set("X-Cypra-Tenant-Role", "admin")
	resp = httptest.NewRecorder()
	router.ServeHTTP(resp, req)
	if resp.Code != http.StatusConflict || !strings.Contains(resp.Body.String(), "member.last_owner") {
		t.Fatalf("remove last owner = %d %s", resp.Code, resp.Body.String())
	}

	req = httptest.NewRequest(http.MethodDelete, "/api/v1/admin/members/"+adminMembershipID.String(), nil)
	req.Host = "acme.cypra.localhost"
	req.Header.Set("X-Cypra-Tenant-Role", "admin")
	resp = httptest.NewRecorder()
	router.ServeHTTP(resp, req)
	if resp.Code != http.StatusNoContent {
		t.Fatalf("remove member = %d %s", resp.Code, resp.Body.String())
	}
	dbtest.RequireCount(t, harness.SQL, `SELECT count(*) FROM tenant_memberships WHERE id = $1`, 0, adminMembershipID)
}

func TestUserDetailActionsUseRealAPIs(t *testing.T) {
	harness := dbtest.New(t)
	tenantID := dbtest.SeedTenant(t, harness.SQL, "acme")
	userID := uuid.New()
	if _, err := harness.SQL.Exec(`INSERT INTO users (id, tenant_id, email, metadata) VALUES ($1, $2, 'ada@example.com', '{}'::jsonb)`, userID, tenantID); err != nil {
		t.Fatalf("seed user: %v", err)
	}
	if _, err := harness.SQL.Exec(`INSERT INTO totp_credentials (tenant_id, user_id, secret_encrypted, confirmed_at) VALUES ($1, $2, 'secret'::bytea, now())`, tenantID, userID); err != nil {
		t.Fatalf("seed totp: %v", err)
	}
	if _, err := harness.SQL.Exec(`INSERT INTO passkey_credentials (tenant_id, user_id, credential_id, public_key, rp_id, purpose) VALUES ($1, $2, 'second-factor'::bytea, 'public'::bytea, 'acme.cypra.localhost', 'second_factor')`, tenantID, userID); err != nil {
		t.Fatalf("seed 2fa passkey: %v", err)
	}
	sessionID := uuid.New()
	if _, err := harness.SQL.Exec(`INSERT INTO sessions (id, subject_id, subject_kind, tenant_id, expires_at) VALUES ($1, $2, 'user', $3, now() + interval '1 hour')`, sessionID, userID, tenantID); err != nil {
		t.Fatalf("seed session: %v", err)
	}
	projectID := uuid.New()
	clientID := uuid.New()
	if _, err := harness.SQL.Exec(`INSERT INTO projects (id, tenant_id, slug, name) VALUES ($1, $2, 'app', 'App')`, projectID, tenantID); err != nil {
		t.Fatalf("seed project: %v", err)
	}
	if _, err := harness.SQL.Exec(`INSERT INTO oidc_clients (id, tenant_id, project_id, client_id, redirect_uris, allowed_scopes, token_endpoint_auth_method) VALUES ($1, $2, $3, 'client_cypra_acme_console', ARRAY['https://app.example/cb'], ARRAY['openid','email'], 'none')`, clientID, tenantID, projectID); err != nil {
		t.Fatalf("seed oidc client: %v", err)
	}
	if _, err := harness.SQL.Exec(`INSERT INTO oidc_consents (tenant_id, user_id, oidc_client_uuid, scopes) VALUES ($1, $2, $3, ARRAY['openid','email'])`, tenantID, userID, clientID); err != nil {
		t.Fatalf("seed consent: %v", err)
	}
	if _, err := harness.SQL.Exec(`INSERT INTO audit_entries (tenant_id, actor_kind, action, resource_kind, resource_id, state_after, metadata) VALUES ($1, 'tenant_admin', 'user.create', 'user', $2, '{}'::jsonb, '{}'::jsonb)`, tenantID, userID); err != nil {
		t.Fatalf("seed audit: %v", err)
	}
	router := newTestServer(t, harness).Router()

	req := httptest.NewRequest(http.MethodGet, "/api/v1/users/"+userID.String(), nil)
	req.Host = "acme.cypra.localhost"
	req.Header.Set("X-Cypra-Tenant-Role", "admin")
	resp := httptest.NewRecorder()
	router.ServeHTTP(resp, req)
	if resp.Code != http.StatusOK || !strings.Contains(resp.Body.String(), "webauthn2fa") || !strings.Contains(resp.Body.String(), "client_cypra_acme_console") || !strings.Contains(resp.Body.String(), sessionID.String()) {
		t.Fatalf("user detail response = %d %s", resp.Code, resp.Body.String())
	}

	for _, path := range []string{"reset-password", "reinvite", "enroll-factor"} {
		req := httptest.NewRequest(http.MethodPost, "/api/v1/users/"+userID.String()+"/"+path, nil)
		req.Host = "acme.cypra.localhost"
		req.Header.Set("X-Cypra-Tenant-Role", "admin")
		resp := httptest.NewRecorder()
		router.ServeHTTP(resp, req)
		if resp.Code != http.StatusOK || !strings.Contains(resp.Body.String(), "token") {
			t.Fatalf("%s response = %d %s", path, resp.Code, resp.Body.String())
		}
	}
	dbtest.RequireCount(t, harness.SQL, `SELECT count(*) FROM pending_invitations WHERE tenant_id = $1 AND email = 'ada@example.com' AND redeemed_at IS NULL`, 1, tenantID)

	req = httptest.NewRequest(http.MethodPost, "/api/v1/users/"+userID.String()+"/disable-mfa", nil)
	req.Host = "acme.cypra.localhost"
	req.Header.Set("X-Cypra-Tenant-Role", "admin")
	resp = httptest.NewRecorder()
	router.ServeHTTP(resp, req)
	if resp.Code != http.StatusOK {
		t.Fatalf("disable mfa response = %d %s", resp.Code, resp.Body.String())
	}
	dbtest.RequireCount(t, harness.SQL, `SELECT count(*) FROM totp_credentials WHERE tenant_id = $1 AND user_id = $2`, 0, tenantID, userID)
	dbtest.RequireCount(t, harness.SQL, `SELECT count(*) FROM passkey_credentials WHERE tenant_id = $1 AND user_id = $2 AND purpose = 'second_factor'`, 0, tenantID, userID)
}

func newTestServer(t *testing.T, harness *dbtest.Harness) *httpserver.Server {
	t.Helper()
	server, err := httpserver.New(httpserver.Options{DB: harness.SQL, TenantDB: harness.TenantDB, PublicBaseURL: "https://cypra.localhost", Version: "test", Commit: "test", DevOpenAPI: true, KEKLoaded: true, MasterKey: bytes.Repeat([]byte{7}, 32), TrustDevHeaders: true})
	if err != nil {
		t.Fatalf("new server: %v", err)
	}
	return server
}

type failingVerifier struct{}

func (failingVerifier) Verify(context.Context, string) error { return errors.New("bot rejected") }
