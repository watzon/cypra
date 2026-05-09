package httpserver_test

import (
	"context"
	"math"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"sort"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/lib/pq"
	"github.com/watzon/cypra/internal/dbtest"
	"github.com/watzon/cypra/internal/httpserver"
	"github.com/watzon/cypra/internal/oidc"
)

func TestPhase18P99PerformanceAssertions(t *testing.T) {
	if os.Getenv("CYPRA_PERF_GATE") != "1" {
		t.Skip("set CYPRA_PERF_GATE=1 to run Phase 18 p99 performance assertions")
	}

	harness := dbtest.New(t)
	tenantID := dbtest.SeedTenant(t, harness.SQL, "acme")
	seedPerformanceUsers(t, harness, tenantID, 10_000)
	seedPerformanceOIDCClient(t, harness, tenantID)

	server, err := httpserver.New(httpserver.Options{DB: harness.SQL, TenantDB: harness.TenantDB, PublicBaseURL: "https://cypra.localhost", Version: "test", Commit: "test", DevOpenAPI: true, KEKLoaded: true, MasterKey: dbtest.TestMasterKey(), TrustDevHeaders: true})
	if err != nil {
		t.Fatalf("new server: %v", err)
	}
	router := server.Router()
	provider := oidc.Provider{DB: harness.SQL, KEK: dbtest.TestMasterKey(), InstallDomain: "cypra.localhost"}
	userID := seedPerformanceOIDCUser(t, harness, tenantID)

	assertP99(t, "/oidc/token", performanceLimitMS("CYPRA_PERF_OIDC_TOKEN_MAX_MS", 200), 40, func() time.Duration {
		verifier := "verifier-" + uuid.NewString()
		code, err := provider.Authorize(context.Background(), oidc.AuthorizeRequest{TenantID: tenantID, UserID: userID, ClientID: "perf-client", RedirectURI: "https://app.example.com/callback", Scope: []string{"openid", "email"}, CodeChallenge: oidcChallenge(verifier), CodeChallengeMethod: "S256", Nonce: "nonce"})
		if err != nil {
			t.Fatalf("authorize perf code: %v", err)
		}
		form := url.Values{"grant_type": {"authorization_code"}, "client_id": {"perf-client"}, "client_secret": {"secret"}, "code": {code}, "redirect_uri": {"https://app.example.com/callback"}, "code_verifier": {verifier}}
		req := httptest.NewRequest(http.MethodPost, "/oidc/token", strings.NewReader(form.Encode()))
		req.Host = "acme.cypra.localhost"
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		resp := httptest.NewRecorder()
		started := time.Now()
		router.ServeHTTP(resp, req)
		elapsed := time.Since(started)
		if resp.Code != http.StatusOK {
			t.Fatalf("token perf status = %d %s", resp.Code, resp.Body.String())
		}
		return elapsed
	})

	assertP99(t, "/login/passkey/verify", performanceLimitMS("CYPRA_PERF_PASSKEY_VERIFY_MAX_MS", 250), 40, func() time.Duration {
		req := httptest.NewRequest(http.MethodPost, "/login/passkey/verify", strings.NewReader(`{}`))
		req.Host = "acme.cypra.localhost"
		req.Header.Set("Content-Type", "application/json")
		resp := httptest.NewRecorder()
		started := time.Now()
		router.ServeHTTP(resp, req)
		elapsed := time.Since(started)
		if resp.Code != http.StatusOK {
			t.Fatalf("passkey perf status = %d %s", resp.Code, resp.Body.String())
		}
		return elapsed
	})

	assertP99(t, "/api/v1/users?limit=50&page=1", performanceLimitMS("CYPRA_PERF_USERS_MAX_MS", 100), 50, func() time.Duration {
		req := httptest.NewRequest(http.MethodGet, "/api/v1/users/?limit=50&page=1", nil)
		req.Host = "acme.cypra.localhost"
		req.Header.Set("X-Cypra-Tenant-Role", "admin")
		resp := httptest.NewRecorder()
		started := time.Now()
		router.ServeHTTP(resp, req)
		elapsed := time.Since(started)
		if resp.Code != http.StatusOK {
			t.Fatalf("users perf status = %d %s", resp.Code, resp.Body.String())
		}
		return elapsed
	})
}

func assertP99(t *testing.T, name string, limit time.Duration, iterations int, exercise func() time.Duration) {
	t.Helper()
	exercise()
	samples := make([]time.Duration, 0, iterations)
	for range iterations {
		samples = append(samples, exercise())
	}
	sort.Slice(samples, func(i, j int) bool { return samples[i] < samples[j] })
	index := int(math.Ceil(float64(len(samples))*0.99)) - 1
	if index < 0 {
		index = 0
	}
	p99 := samples[index]
	t.Logf("%s p99=%s limit=%s samples=%d", name, p99, limit, len(samples))
	if p99 > limit {
		t.Fatalf("%s p99 %s exceeds %s", name, p99, limit)
	}
}

func performanceLimitMS(env string, fallback int) time.Duration {
	if raw := os.Getenv(env); raw != "" {
		if parsed, err := strconv.Atoi(raw); err == nil && parsed > 0 {
			return time.Duration(parsed) * time.Millisecond
		}
	}
	return time.Duration(fallback) * time.Millisecond
}

func seedPerformanceUsers(t *testing.T, harness *dbtest.Harness, tenantID uuid.UUID, count int) {
	t.Helper()
	tx, err := harness.SQL.Begin()
	if err != nil {
		t.Fatalf("begin seed perf users: %v", err)
	}
	defer func() { _ = tx.Rollback() }()
	stmt, err := tx.Prepare(`INSERT INTO users (id, tenant_id, email, metadata) VALUES ($1, $2, $3, '{}'::jsonb)`)
	if err != nil {
		t.Fatalf("prepare seed perf users: %v", err)
	}
	defer func() { _ = stmt.Close() }()
	for i := range count {
		if _, err := stmt.Exec(uuid.New(), tenantID, "perf-"+strconv.Itoa(100_000+i)+"@example.com"); err != nil {
			t.Fatalf("seed perf user %d: %v", i, err)
		}
	}
	if err := tx.Commit(); err != nil {
		t.Fatalf("commit seed perf users: %v", err)
	}
}

func seedPerformanceOIDCUser(t *testing.T, harness *dbtest.Harness, tenantID uuid.UUID) uuid.UUID {
	t.Helper()
	userID := uuid.New()
	if _, err := harness.SQL.Exec(`INSERT INTO users (id, tenant_id, email, email_verified_at, metadata) VALUES ($1, $2, 'oidc-perf@example.com', now(), '{}'::jsonb)`, userID, tenantID); err != nil {
		t.Fatalf("seed oidc perf user: %v", err)
	}
	return userID
}

func seedPerformanceOIDCClient(t *testing.T, harness *dbtest.Harness, tenantID uuid.UUID) {
	t.Helper()
	projectID := uuid.New()
	if _, err := harness.SQL.Exec(`INSERT INTO projects (id, tenant_id, slug, name) VALUES ($1, $2, 'perf-app', 'Perf App')`, projectID, tenantID); err != nil {
		t.Fatalf("seed perf project: %v", err)
	}
	secret, err := oidc.EncryptClientSecret("secret", dbtest.TestMasterKey())
	if err != nil {
		t.Fatalf("encrypt client secret: %v", err)
	}
	if _, err := harness.SQL.Exec(`INSERT INTO oidc_clients (tenant_id, project_id, client_id, client_secret_encrypted, redirect_uris, allowed_scopes, token_endpoint_auth_method) VALUES ($1, $2, 'perf-client', $3, $4, $5, 'client_secret_post')`, tenantID, projectID, secret, pq.Array([]string{"https://app.example.com/callback"}), pq.Array([]string{"openid", "email"})); err != nil {
		t.Fatalf("seed perf client: %v", err)
	}
}
